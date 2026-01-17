package service

import "time"

type Group struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Description    string  `json:"description,omitempty"`
	Platform       string  `json:"platform"`
	RateMultiplier float64 `json:"rate_multiplier"`
	IsExclusive    bool    `json:"is_exclusive"`
	Status         string  `json:"status"`
	Hydrated       bool    `json:"-"` // indicates the group was loaded from a trusted repository source

	SubscriptionType    string   `json:"subscription_type"`
	DailyLimitUSD       *float64 `json:"daily_limit_usd,omitempty"`
	WeeklyLimitUSD      *float64 `json:"weekly_limit_usd,omitempty"`
	MonthlyLimitUSD     *float64 `json:"monthly_limit_usd,omitempty"`
	DefaultValidityDays int      `json:"default_validity_days"`

	// 图片生成计费配置（antigravity 和 gemini 平台使用）
	ImagePrice1K *float64 `json:"image_price_1k,omitempty"`
	ImagePrice2K *float64 `json:"image_price_2k,omitempty"`
	ImagePrice4K *float64 `json:"image_price_4k,omitempty"`

	// Claude Code 客户端限制
	ClaudeCodeOnly  bool   `json:"claude_code_only"`
	FallbackGroupID *int64 `json:"fallback_group_id,omitempty"`

	// 余额计费模式限额
	BalanceDailyQuota  *float64 `json:"balance_daily_quota,omitempty"`
	BalanceWeeklyQuota *float64 `json:"balance_weekly_quota,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	AccountGroups []AccountGroup `json:"account_groups,omitempty"`
	AccountCount  int64          `json:"account_count"`
}

func (g *Group) IsActive() bool {
	return g.Status == StatusActive
}

func (g *Group) IsSubscriptionType() bool {
	return g.SubscriptionType == SubscriptionTypeSubscription
}

func (g *Group) IsFreeSubscription() bool {
	return g.IsSubscriptionType() && g.RateMultiplier == 0
}

func (g *Group) HasDailyLimit() bool {
	return g.DailyLimitUSD != nil && *g.DailyLimitUSD > 0
}

func (g *Group) HasWeeklyLimit() bool {
	return g.WeeklyLimitUSD != nil && *g.WeeklyLimitUSD > 0
}

func (g *Group) HasMonthlyLimit() bool {
	return g.MonthlyLimitUSD != nil && *g.MonthlyLimitUSD > 0
}

// HasBalanceDailyQuota 检查是否配置了余额计费每日限额
func (g *Group) HasBalanceDailyQuota() bool {
	return g.BalanceDailyQuota != nil && *g.BalanceDailyQuota > 0
}

// HasBalanceWeeklyQuota 检查是否配置了余额计费每周限额
func (g *Group) HasBalanceWeeklyQuota() bool {
	return g.BalanceWeeklyQuota != nil && *g.BalanceWeeklyQuota > 0
}

// GetImagePrice 根据 image_size 返回对应的图片生成价格
// 如果分组未配置价格，返回 nil（调用方应使用默认值）
func (g *Group) GetImagePrice(imageSize string) *float64 {
	switch imageSize {
	case "1K":
		return g.ImagePrice1K
	case "2K":
		return g.ImagePrice2K
	case "4K":
		return g.ImagePrice4K
	default:
		// 未知尺寸默认按 2K 计费
		return g.ImagePrice2K
	}
}

// IsGroupContextValid reports whether a group from context has the fields required for routing decisions.
func IsGroupContextValid(group *Group) bool {
	if group == nil {
		return false
	}
	if group.ID <= 0 {
		return false
	}
	if !group.Hydrated {
		return false
	}
	if group.Platform == "" || group.Status == "" {
		return false
	}
	return true
}
