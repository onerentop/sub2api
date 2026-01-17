package service

import (
	"time"
)

// AnnouncementType 公告类型
type AnnouncementType string

const (
	AnnouncementTypeInfo    AnnouncementType = "info"
	AnnouncementTypeSuccess AnnouncementType = "success"
	AnnouncementTypeWarning AnnouncementType = "warning"
	AnnouncementTypeError   AnnouncementType = "error"
)

// Announcement 公告实体
type Announcement struct {
	ID        int64            `json:"id"`
	Title     string           `json:"title"`
	Content   string           `json:"content"`
	Type      AnnouncementType `json:"type"`
	SortOrder int              `json:"sort_order"`
	Enabled   bool             `json:"enabled"`
	StartTime *time.Time       `json:"start_time,omitempty"`
	EndTime   *time.Time       `json:"end_time,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt *time.Time       `json:"-"`
}

// IsActive 检查公告是否当前有效
func (a *Announcement) IsActive() bool {
	if !a.Enabled {
		return false
	}
	if a.DeletedAt != nil {
		return false
	}
	now := time.Now()
	if a.StartTime != nil && now.Before(*a.StartTime) {
		return false
	}
	if a.EndTime != nil && now.After(*a.EndTime) {
		return false
	}
	return true
}

// IsScheduled 检查是否为定时公告（尚未生效）
func (a *Announcement) IsScheduled() bool {
	if a.StartTime == nil {
		return false
	}
	return time.Now().Before(*a.StartTime)
}

// IsExpired 检查是否已过期
func (a *Announcement) IsExpired() bool {
	if a.EndTime == nil {
		return false
	}
	return time.Now().After(*a.EndTime)
}

// CreateAnnouncementInput 创建公告输入
type CreateAnnouncementInput struct {
	Title     string
	Content   string
	Type      AnnouncementType
	Enabled   bool
	StartTime *time.Time
	EndTime   *time.Time
}

// UpdateAnnouncementInput 更新公告输入
type UpdateAnnouncementInput struct {
	Title     *string
	Content   *string
	Type      *AnnouncementType
	SortOrder *int
	Enabled   *bool
	StartTime *time.Time
	EndTime   *time.Time
	// ClearStartTime 用于清除生效时间
	ClearStartTime bool
	// ClearEndTime 用于清除过期时间
	ClearEndTime bool
}

// AnnouncementSortItem 排序项
type AnnouncementSortItem struct {
	ID        int64 `json:"id"`
	SortOrder int   `json:"sort_order"`
}

// ActiveAnnouncementsResponse 活动公告响应
type ActiveAnnouncementsResponse struct {
	Announcements []Announcement       `json:"announcements"`
	Settings      AnnouncementSettings `json:"settings"`
}

// AnnouncementSettings 公告设置
type AnnouncementSettings struct {
	Enabled  bool `json:"enabled"`
	Interval int  `json:"interval"` // 轮播间隔（毫秒）
}
