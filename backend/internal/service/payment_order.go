package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// 支付相关错误定义
var (
	ErrProductNotFound      = infraerrors.NotFound("PRODUCT_NOT_FOUND", "product not found")
	ErrPaymentOrderNotFound = infraerrors.NotFound("PAYMENT_ORDER_NOT_FOUND", "payment order not found")
	ErrPaymentOrderPending  = infraerrors.Conflict("PAYMENT_ORDER_PENDING", "payment order is pending")
	ErrPaymentOrderPaid     = infraerrors.Conflict("PAYMENT_ORDER_PAID", "payment order already paid")
	ErrPaymentAmountInvalid = infraerrors.BadRequest("PAYMENT_AMOUNT_INVALID", "payment amount is invalid")
	ErrPaymentSignInvalid   = infraerrors.BadRequest("PAYMENT_SIGN_INVALID", "payment signature is invalid")
)

// PaymentOrder 支付订单领域模型
type PaymentOrder struct {
	ID            int64          `json:"id"`
	UserID        int64          `json:"user_id"`
	ProductID     *int64         `json:"product_id,omitempty"`
	OrderNo       string         `json:"order_no"`
	TradeNo       *string        `json:"trade_no,omitempty"`
	AmountCNY     float64        `json:"amount_cny"`     // 支付金额（人民币）
	AmountValue   float64        `json:"amount_value"`   // 到账价值（余额USD或订阅天数）
	OrderType     string         `json:"order_type"`     // balance 或 subscription
	PaymentMethod string         `json:"payment_method"` // wechat 或 alipay
	Status        string         `json:"status"`         // pending/paid/failed/refunded/auditing
	PaidAt        *time.Time     `json:"paid_at,omitempty"`
	CallbackData  map[string]any `json:"-"` // 不返回给前端
	Remark        *string        `json:"remark,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`

	// 关联数据
	User    *User    `json:"user,omitempty"`
	Product *Product `json:"product,omitempty"`
}

// IsPending 是否为待支付状态
func (o *PaymentOrder) IsPending() bool {
	return o.Status == PaymentOrderStatusPending
}

// IsPaid 是否已支付
func (o *PaymentOrder) IsPaid() bool {
	return o.Status == PaymentOrderStatusPaid
}

// IsAuditing 是否在审核中
func (o *PaymentOrder) IsAuditing() bool {
	return o.Status == PaymentOrderStatusAuditing
}

// CanPay 是否可以支付
func (o *PaymentOrder) CanPay() bool {
	return o.Status == PaymentOrderStatusPending
}

// PaymentOrderStats 订单统计数据
type PaymentOrderStats struct {
	TotalOrders   int     `json:"total_orders"`
	TotalAmount   float64 `json:"total_amount"`
	PaidOrders    int     `json:"paid_orders"`
	PaidAmount    float64 `json:"paid_amount"`
	PendingOrders int     `json:"pending_orders"`
	PendingAmount float64 `json:"pending_amount"`
	TodayOrders   int     `json:"today_orders"`
	TodayAmount   float64 `json:"today_amount"`
}

// PaymentOrderRepository 支付订单数据访问接口
type PaymentOrderRepository interface {
	Create(ctx context.Context, order *PaymentOrder) error
	GetByID(ctx context.Context, id int64) (*PaymentOrder, error)
	GetByOrderNo(ctx context.Context, orderNo string) (*PaymentOrder, error)
	ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]PaymentOrder, *pagination.PaginationResult, error)
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, status, orderType, paymentMethod, search string) ([]PaymentOrder, *pagination.PaginationResult, error)
	Update(ctx context.Context, order *PaymentOrder) error
	UpdateStatus(ctx context.Context, id int64, status string) error
	MarkAsPaid(ctx context.Context, id int64, tradeNo string, callbackData map[string]any) error
	GetOrderStats(ctx context.Context) (*PaymentOrderStats, error)
}

// 订单状态常量
const (
	PaymentOrderStatusPending  = "pending"  // 待支付
	PaymentOrderStatusPaid     = "paid"     // 已支付
	PaymentOrderStatusFailed   = "failed"   // 支付失败
	PaymentOrderStatusRefunded = "refunded" // 已退款
	PaymentOrderStatusAuditing = "auditing" // 审核中
)

// 支付方式常量
const (
	PaymentMethodWeChat = "wechat" // 微信支付
	PaymentMethodAlipay = "alipay" // 支付宝
)
