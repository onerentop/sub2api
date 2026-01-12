package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// AnnouncementRepository 公告仓储接口
type AnnouncementRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, announcement *Announcement) error
	GetByID(ctx context.Context, id int64) (*Announcement, error)
	Update(ctx context.Context, announcement *Announcement) error
	Delete(ctx context.Context, id int64) error

	// 列表查询
	List(ctx context.Context, params pagination.PaginationParams) ([]Announcement, *pagination.PaginationResult, error)
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, enabled *bool, announcementType string) ([]Announcement, *pagination.PaginationResult, error)

	// 获取当前生效的公告（用于前端展示）
	ListActive(ctx context.Context) ([]Announcement, error)

	// 批量更新排序
	UpdateSortOrders(ctx context.Context, items []AnnouncementSortItem) error

	// 获取下一个排序值
	GetMaxSortOrder(ctx context.Context) (int, error)
}
