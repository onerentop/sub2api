package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/product"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type productRepository struct {
	client *dbent.Client
}

// NewProductRepository creates a new product repository
func NewProductRepository(client *dbent.Client) service.ProductRepository {
	return &productRepository{client: client}
}

func (r *productRepository) Create(ctx context.Context, p *service.Product) error {
	builder := r.client.Product.Create().
		SetName(p.Name).
		SetType(product.Type(p.Type)).
		SetPriceCny(p.PriceCNY).
		SetValue(p.Value).
		SetIsActive(p.IsActive).
		SetSortOrder(p.SortOrder)

	if p.Description != nil {
		builder.SetDescription(*p.Description)
	}
	if p.GroupID != nil {
		builder.SetGroupID(*p.GroupID)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	p.ID = created.ID
	p.CreatedAt = created.CreatedAt
	p.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *productRepository) GetByID(ctx context.Context, id int64) (*service.Product, error) {
	m, err := r.client.Product.Query().
		Where(product.IDEQ(id), product.DeletedAtIsNil()).
		WithGroup().
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrProductNotFound
		}
		return nil, err
	}
	return productEntityToService(m), nil
}

func (r *productRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.Product, *pagination.PaginationResult, error) {
	return r.ListWithFilters(ctx, params, "", nil, "")
}

func (r *productRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, productType string, isActive *bool, search string) ([]service.Product, *pagination.PaginationResult, error) {
	q := r.client.Product.Query().Where(product.DeletedAtIsNil())

	if productType != "" {
		q = q.Where(product.TypeEQ(product.Type(productType)))
	}
	if isActive != nil {
		q = q.Where(product.IsActiveEQ(*isActive))
	}
	if search != "" {
		q = q.Where(product.NameContainsFold(search))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	products, err := q.
		WithGroup().
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Asc(product.FieldSortOrder), dbent.Desc(product.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return productEntitiesToService(products), paginationResultFromTotal(int64(total), params), nil
}

func (r *productRepository) ListActive(ctx context.Context) ([]service.Product, error) {
	products, err := r.client.Product.Query().
		Where(product.DeletedAtIsNil(), product.IsActiveEQ(true)).
		WithGroup().
		Order(dbent.Asc(product.FieldSortOrder), dbent.Desc(product.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return productEntitiesToService(products), nil
}

func (r *productRepository) Update(ctx context.Context, p *service.Product) error {
	builder := r.client.Product.UpdateOneID(p.ID).
		SetName(p.Name).
		SetType(product.Type(p.Type)).
		SetPriceCny(p.PriceCNY).
		SetValue(p.Value).
		SetIsActive(p.IsActive).
		SetSortOrder(p.SortOrder)

	if p.Description != nil {
		builder.SetDescription(*p.Description)
	} else {
		builder.ClearDescription()
	}
	if p.GroupID != nil {
		builder.SetGroupID(*p.GroupID)
	} else {
		builder.ClearGroupID()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrProductNotFound
		}
		return err
	}
	p.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *productRepository) Delete(ctx context.Context, id int64) error {
	// Soft delete
	affected, err := r.client.Product.Update().
		Where(product.IDEQ(id), product.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrProductNotFound
	}
	return nil
}

func productEntityToService(m *dbent.Product) *service.Product {
	if m == nil {
		return nil
	}
	p := &service.Product{
		ID:        m.ID,
		Name:      m.Name,
		Type:      string(m.Type),
		PriceCNY:  m.PriceCny,
		Value:     m.Value,
		GroupID:   m.GroupID,
		IsActive:  m.IsActive,
		SortOrder: m.SortOrder,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.Description != nil {
		p.Description = m.Description
	}
	if m.Edges.Group != nil {
		p.Group = groupEntityToService(m.Edges.Group)
	}
	return p
}

func productEntitiesToService(models []*dbent.Product) []service.Product {
	out := make([]service.Product, 0, len(models))
	for _, m := range models {
		if p := productEntityToService(m); p != nil {
			out = append(out, *p)
		}
	}
	return out
}
