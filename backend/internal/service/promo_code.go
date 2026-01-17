package service

import (
	"time"
)

// PromoCode 注册优惠码
type PromoCode struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	BonusAmount float64    `json:"bonus_amount"`
	MaxUses     int        `json:"max_uses"`
	UsedCount   int        `json:"used_count"`
	Status      string     `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Notes       string     `json:"notes,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// 关联
	UsageRecords []PromoCodeUsage `json:"usage_records,omitempty"`
}

// PromoCodeUsage 优惠码使用记录
type PromoCodeUsage struct {
	ID          int64     `json:"id"`
	PromoCodeID int64     `json:"promo_code_id"`
	UserID      int64     `json:"user_id"`
	BonusAmount float64   `json:"bonus_amount"`
	UsedAt      time.Time `json:"used_at"`

	// 关联
	PromoCode *PromoCode `json:"promo_code,omitempty"`
	User      *User      `json:"user,omitempty"`
}

// CanUse 检查优惠码是否可用
func (p *PromoCode) CanUse() bool {
	if p.Status != PromoCodeStatusActive {
		return false
	}
	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		return false
	}
	if p.MaxUses > 0 && p.UsedCount >= p.MaxUses {
		return false
	}
	return true
}

// IsExpired 检查是否已过期
func (p *PromoCode) IsExpired() bool {
	return p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt)
}

// CreatePromoCodeInput 创建优惠码输入
type CreatePromoCodeInput struct {
	Code        string
	BonusAmount float64
	MaxUses     int
	ExpiresAt   *time.Time
	Notes       string
}

// UpdatePromoCodeInput 更新优惠码输入
type UpdatePromoCodeInput struct {
	Code        *string
	BonusAmount *float64
	MaxUses     *int
	Status      *string
	ExpiresAt   *time.Time
	Notes       *string
}
