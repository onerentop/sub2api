package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/announcement"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type announcementRepository struct {
	client *dbent.Client
}

// NewAnnouncementRepository creates a new announcement repository
func NewAnnouncementRepository(client *dbent.Client) service.AnnouncementRepository {
	return &announcementRepository{client: client}
}

func (r *announcementRepository) Create(ctx context.Context, a *service.Announcement) error {
	client := clientFromContext(ctx, r.client)
	builder := client.Announcement.Create().
		SetContent(a.Content).
		SetType(announcement.Type(a.Type)).
		SetSortOrder(a.SortOrder).
		SetEnabled(a.Enabled)

	if a.Title != "" {
		builder.SetTitle(a.Title)
	}
	if a.StartTime != nil {
		builder.SetStartTime(*a.StartTime)
	}
	if a.EndTime != nil {
		builder.SetEndTime(*a.EndTime)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}

	a.ID = created.ID
	a.CreatedAt = created.CreatedAt
	a.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *announcementRepository) GetByID(ctx context.Context, id int64) (*service.Announcement, error) {
	m, err := r.client.Announcement.Query().
		Where(
			announcement.IDEQ(id),
			announcement.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrAnnouncementNotFound
		}
		return nil, err
	}
	return announcementEntityToService(m), nil
}

func (r *announcementRepository) Update(ctx context.Context, a *service.Announcement) error {
	client := clientFromContext(ctx, r.client)
	builder := client.Announcement.UpdateOneID(a.ID).
		SetContent(a.Content).
		SetType(announcement.Type(a.Type)).
		SetSortOrder(a.SortOrder).
		SetEnabled(a.Enabled)

	if a.Title != "" {
		builder.SetTitle(a.Title)
	} else {
		builder.ClearTitle()
	}

	if a.StartTime != nil {
		builder.SetStartTime(*a.StartTime)
	} else {
		builder.ClearStartTime()
	}

	if a.EndTime != nil {
		builder.SetEndTime(*a.EndTime)
	} else {
		builder.ClearEndTime()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrAnnouncementNotFound
		}
		return err
	}

	a.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *announcementRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	// 软删除
	now := time.Now()
	_, err := client.Announcement.UpdateOneID(id).
		SetDeletedAt(now).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrAnnouncementNotFound
		}
		return err
	}
	return nil
}

func (r *announcementRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.Announcement, *pagination.PaginationResult, error) {
	return r.ListWithFilters(ctx, params, nil, "")
}

func (r *announcementRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, enabled *bool, announcementType string) ([]service.Announcement, *pagination.PaginationResult, error) {
	q := r.client.Announcement.Query().
		Where(announcement.DeletedAtIsNil())

	if enabled != nil {
		q = q.Where(announcement.EnabledEQ(*enabled))
	}
	if announcementType != "" {
		q = q.Where(announcement.TypeEQ(announcement.Type(announcementType)))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	items, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Asc(announcement.FieldSortOrder), dbent.Desc(announcement.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	out := announcementEntitiesToService(items)
	return out, paginationResultFromTotal(int64(total), params), nil
}

func (r *announcementRepository) ListActive(ctx context.Context) ([]service.Announcement, error) {
	now := time.Now()
	items, err := r.client.Announcement.Query().
		Where(
			announcement.DeletedAtIsNil(),
			announcement.EnabledEQ(true),
			// start_time is null OR start_time <= now
			announcement.Or(
				announcement.StartTimeIsNil(),
				announcement.StartTimeLTE(now),
			),
			// end_time is null OR end_time > now
			announcement.Or(
				announcement.EndTimeIsNil(),
				announcement.EndTimeGT(now),
			),
		).
		Order(dbent.Asc(announcement.FieldSortOrder), dbent.Desc(announcement.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return announcementEntitiesToService(items), nil
}

func (r *announcementRepository) UpdateSortOrders(ctx context.Context, items []service.AnnouncementSortItem) error {
	client := clientFromContext(ctx, r.client)
	for _, item := range items {
		_, err := client.Announcement.UpdateOneID(item.ID).
			SetSortOrder(item.SortOrder).
			Save(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *announcementRepository) GetMaxSortOrder(ctx context.Context) (int, error) {
	m, err := r.client.Announcement.Query().
		Where(announcement.DeletedAtIsNil()).
		Order(dbent.Desc(announcement.FieldSortOrder)).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	return m.SortOrder, nil
}

// Entity to Service conversions

func announcementEntityToService(m *dbent.Announcement) *service.Announcement {
	if m == nil {
		return nil
	}
	return &service.Announcement{
		ID:        m.ID,
		Title:     derefString(m.Title),
		Content:   m.Content,
		Type:      service.AnnouncementType(m.Type),
		SortOrder: m.SortOrder,
		Enabled:   m.Enabled,
		StartTime: m.StartTime,
		EndTime:   m.EndTime,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func announcementEntitiesToService(models []*dbent.Announcement) []service.Announcement {
	out := make([]service.Announcement, 0, len(models))
	for i := range models {
		if s := announcementEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
