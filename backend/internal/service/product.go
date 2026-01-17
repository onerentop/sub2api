package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// Product 商品领域模型
type Product struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Type        string     `json:"type"`      // balance 或 subscription
	PriceCNY    float64    `json:"price_cny"` // 人民币价格
	Value       float64    `json:"value"`     // 价值：余额(USD) 或 订阅天数
	GroupID     *int64     `json:"group_id,omitempty"`
	IsActive    bool       `json:"is_active"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"-"` // 不返回给前端

	// 关联数据
	Group *Group `json:"group,omitempty"`
}

// IsBalanceType 是否为余额充值类型
func (p *Product) IsBalanceType() bool {
	return p.Type == ProductTypeBalance
}

// IsSubscriptionType 是否为订阅套餐类型
func (p *Product) IsSubscriptionType() bool {
	return p.Type == ProductTypeSubscription
}

// ProductRepository 商品数据访问接口
type ProductRepository interface {
	Create(ctx context.Context, p *Product) error
	GetByID(ctx context.Context, id int64) (*Product, error)
	List(ctx context.Context, params pagination.PaginationParams) ([]Product, *pagination.PaginationResult, error)
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, productType string, isActive *bool, search string) ([]Product, *pagination.PaginationResult, error)
	ListActive(ctx context.Context) ([]Product, error)
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id int64) error
}

// 商品类型常量
const (
	ProductTypeBalance      = "balance"      // 余额充值
	ProductTypeSubscription = "subscription" // 订阅套餐
)
