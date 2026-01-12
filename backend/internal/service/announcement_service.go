package service

import (
	"context"
	"fmt"
	"strconv"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrAnnouncementNotFound = infraerrors.NotFound("ANNOUNCEMENT_NOT_FOUND", "announcement not found")
)

// AnnouncementService 公告服务
type AnnouncementService struct {
	announcementRepo AnnouncementRepository
	settingRepo      SettingRepository
}

// NewAnnouncementService creates a new announcement service
func NewAnnouncementService(
	announcementRepo AnnouncementRepository,
	settingRepo SettingRepository,
) *AnnouncementService {
	return &AnnouncementService{
		announcementRepo: announcementRepo,
		settingRepo:      settingRepo,
	}
}

// Create 创建公告
func (s *AnnouncementService) Create(ctx context.Context, input *CreateAnnouncementInput) (*Announcement, error) {
	// 获取下一个排序值
	maxSortOrder, err := s.announcementRepo.GetMaxSortOrder(ctx)
	if err != nil {
		return nil, fmt.Errorf("get max sort order: %w", err)
	}

	announcement := &Announcement{
		Title:     input.Title,
		Content:   input.Content,
		Type:      input.Type,
		SortOrder: maxSortOrder + 1,
		Enabled:   input.Enabled,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
	}

	if announcement.Type == "" {
		announcement.Type = AnnouncementTypeInfo
	}

	if err := s.announcementRepo.Create(ctx, announcement); err != nil {
		return nil, fmt.Errorf("create announcement: %w", err)
	}

	return announcement, nil
}

// GetByID 根据ID获取公告
func (s *AnnouncementService) GetByID(ctx context.Context, id int64) (*Announcement, error) {
	return s.announcementRepo.GetByID(ctx, id)
}

// Update 更新公告
func (s *AnnouncementService) Update(ctx context.Context, id int64, input *UpdateAnnouncementInput) (*Announcement, error) {
	announcement, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		announcement.Title = *input.Title
	}
	if input.Content != nil {
		announcement.Content = *input.Content
	}
	if input.Type != nil {
		announcement.Type = *input.Type
	}
	if input.SortOrder != nil {
		announcement.SortOrder = *input.SortOrder
	}
	if input.Enabled != nil {
		announcement.Enabled = *input.Enabled
	}
	if input.ClearStartTime {
		announcement.StartTime = nil
	} else if input.StartTime != nil {
		announcement.StartTime = input.StartTime
	}
	if input.ClearEndTime {
		announcement.EndTime = nil
	} else if input.EndTime != nil {
		announcement.EndTime = input.EndTime
	}

	if err := s.announcementRepo.Update(ctx, announcement); err != nil {
		return nil, fmt.Errorf("update announcement: %w", err)
	}

	return announcement, nil
}

// Delete 删除公告（软删除）
func (s *AnnouncementService) Delete(ctx context.Context, id int64) error {
	if err := s.announcementRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	return nil
}

// List 获取公告列表
func (s *AnnouncementService) List(ctx context.Context, params pagination.PaginationParams, enabled *bool, announcementType string) ([]Announcement, *pagination.PaginationResult, error) {
	return s.announcementRepo.ListWithFilters(ctx, params, enabled, announcementType)
}

// UpdateSortOrders 批量更新排序
func (s *AnnouncementService) UpdateSortOrders(ctx context.Context, items []AnnouncementSortItem) error {
	return s.announcementRepo.UpdateSortOrders(ctx, items)
}

// GetActiveAnnouncements 获取当前生效的公告（用于前端展示）
func (s *AnnouncementService) GetActiveAnnouncements(ctx context.Context) (*ActiveAnnouncementsResponse, error) {
	// 检查全局开关
	settings, err := s.getAnnouncementSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("get announcement settings: %w", err)
	}

	if !settings.Enabled {
		return &ActiveAnnouncementsResponse{
			Announcements: []Announcement{},
			Settings:      settings,
		}, nil
	}

	announcements, err := s.announcementRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active announcements: %w", err)
	}

	return &ActiveAnnouncementsResponse{
		Announcements: announcements,
		Settings:      settings,
	}, nil
}

// getAnnouncementSettings 获取公告相关设置
func (s *AnnouncementService) getAnnouncementSettings(ctx context.Context) (AnnouncementSettings, error) {
	settings := AnnouncementSettings{
		Enabled:  true,
		Interval: 5000,
	}

	// 获取全局开关
	enabledStr, err := s.settingRepo.GetValue(ctx, "announcement.enabled")
	if err == nil && enabledStr != "" {
		settings.Enabled = enabledStr == "true"
	}

	// 获取轮播间隔
	intervalStr, err := s.settingRepo.GetValue(ctx, "announcement.interval")
	if err == nil && intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil && interval > 0 {
			settings.Interval = interval
		}
	}

	return settings, nil
}
