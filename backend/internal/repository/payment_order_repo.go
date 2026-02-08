package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type paymentOrderRepository struct {
	client *dbent.Client
}

// NewPaymentOrderRepository creates a new payment order repository
func NewPaymentOrderRepository(client *dbent.Client) service.PaymentOrderRepository {
	return &paymentOrderRepository{client: client}
}

func (r *paymentOrderRepository) Create(ctx context.Context, order *service.PaymentOrder) error {
	builder := r.client.PaymentOrder.Create().
		SetUserID(order.UserID).
		SetOrderNo(order.OrderNo).
		SetAmountCny(order.AmountCNY).
		SetAmountValue(order.AmountValue).
		SetOrderType(paymentorder.OrderType(order.OrderType)).
		SetPaymentMethod(paymentorder.PaymentMethod(order.PaymentMethod)).
		SetStatus(paymentorder.Status(order.Status))

	if order.ProductID != nil {
		builder.SetProductID(*order.ProductID)
	}
	if order.TradeNo != nil {
		builder.SetTradeNo(*order.TradeNo)
	}
	if order.PaidAt != nil {
		builder.SetPaidAt(*order.PaidAt)
	}
	if order.CallbackData != nil {
		builder.SetCallbackData(order.CallbackData)
	}
	if order.Remark != nil {
		builder.SetRemark(*order.Remark)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	order.ID = created.ID
	order.CreatedAt = created.CreatedAt
	order.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *paymentOrderRepository) GetByID(ctx context.Context, id int64) (*service.PaymentOrder, error) {
	m, err := r.client.PaymentOrder.Query().
		Where(paymentorder.IDEQ(id)).
		WithUser().
		WithProduct().
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrPaymentOrderNotFound
		}
		return nil, err
	}
	return paymentOrderEntityToService(m), nil
}

func (r *paymentOrderRepository) GetByOrderNo(ctx context.Context, orderNo string) (*service.PaymentOrder, error) {
	m, err := r.client.PaymentOrder.Query().
		Where(paymentorder.OrderNoEQ(orderNo)).
		WithUser().
		WithProduct().
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrPaymentOrderNotFound
		}
		return nil, err
	}
	return paymentOrderEntityToService(m), nil
}

func (r *paymentOrderRepository) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.PaymentOrder, *pagination.PaginationResult, error) {
	q := r.client.PaymentOrder.Query().Where(paymentorder.UserIDEQ(userID))

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	orders, err := q.
		WithProduct().
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(paymentorder.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return paymentOrderEntitiesToService(orders), paginationResultFromTotal(int64(total), params), nil
}

func (r *paymentOrderRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, status, orderType, paymentMethod, search string) ([]service.PaymentOrder, *pagination.PaginationResult, error) {
	q := r.client.PaymentOrder.Query()

	if status != "" {
		q = q.Where(paymentorder.StatusEQ(paymentorder.Status(status)))
	}
	if orderType != "" {
		q = q.Where(paymentorder.OrderTypeEQ(paymentorder.OrderType(orderType)))
	}
	if paymentMethod != "" {
		q = q.Where(paymentorder.PaymentMethodEQ(paymentorder.PaymentMethod(paymentMethod)))
	}
	if search != "" {
		q = q.Where(paymentorder.OrderNoContainsFold(search))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	orders, err := q.
		WithUser().
		WithProduct().
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(paymentorder.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return paymentOrderEntitiesToService(orders), paginationResultFromTotal(int64(total), params), nil
}

func (r *paymentOrderRepository) Update(ctx context.Context, order *service.PaymentOrder) error {
	builder := r.client.PaymentOrder.UpdateOneID(order.ID).
		SetStatus(paymentorder.Status(order.Status))

	if order.TradeNo != nil {
		builder.SetTradeNo(*order.TradeNo)
	}
	if order.PaidAt != nil {
		builder.SetPaidAt(*order.PaidAt)
	}
	if order.CallbackData != nil {
		builder.SetCallbackData(order.CallbackData)
	}
	if order.Remark != nil {
		builder.SetRemark(*order.Remark)
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrPaymentOrderNotFound
		}
		return err
	}
	order.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *paymentOrderRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	affected, err := r.client.PaymentOrder.Update().
		Where(paymentorder.IDEQ(id)).
		SetStatus(paymentorder.Status(status)).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPaymentOrderNotFound
	}
	return nil
}

func (r *paymentOrderRepository) MarkAsPaid(ctx context.Context, id int64, tradeNo string, callbackData map[string]any) error {
	now := time.Now()
	affected, err := r.client.PaymentOrder.Update().
		Where(paymentorder.IDEQ(id), paymentorder.StatusEQ(paymentorder.StatusPending)).
		SetStatus(paymentorder.StatusPaid).
		SetTradeNo(tradeNo).
		SetPaidAt(now).
		SetCallbackData(callbackData).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPaymentOrderNotFound
	}
	return nil
}

func (r *paymentOrderRepository) GetOrderStats(ctx context.Context) (*service.PaymentOrderStats, error) {
	result := &service.PaymentOrderStats{}

	// Query stats grouped by status
	var stats []struct {
		Status string  `json:"status"`
		Count  int     `json:"count"`
		Sum    float64 `json:"sum"`
	}

	err := r.client.PaymentOrder.Query().
		GroupBy(paymentorder.FieldStatus).
		Aggregate(
			dbent.Count(),
			dbent.Sum(paymentorder.FieldAmountCny),
		).
		Scan(ctx, &stats)
	if err != nil {
		return nil, err
	}

	for _, s := range stats {
		result.TotalOrders += s.Count
		result.TotalAmount += s.Sum
		switch s.Status {
		case "paid":
			result.PaidOrders = s.Count
			result.PaidAmount = s.Sum
		case "pending":
			result.PendingOrders = s.Count
			result.PendingAmount = s.Sum
		case "auditing":
			result.PendingOrders += s.Count
			result.PendingAmount += s.Sum
		}
	}

	// Query today's orders
	today := time.Now().Truncate(24 * time.Hour)
	todayOrders, err := r.client.PaymentOrder.Query().
		Where(paymentorder.CreatedAtGTE(today)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	for _, order := range todayOrders {
		result.TodayOrders++
		result.TodayAmount += order.AmountCny
	}

	return result, nil
}

func paymentOrderEntityToService(m *dbent.PaymentOrder) *service.PaymentOrder {
	if m == nil {
		return nil
	}
	order := &service.PaymentOrder{
		ID:            m.ID,
		UserID:        m.UserID,
		ProductID:     m.ProductID,
		OrderNo:       m.OrderNo,
		TradeNo:       m.TradeNo,
		AmountCNY:     m.AmountCny,
		AmountValue:   m.AmountValue,
		OrderType:     string(m.OrderType),
		PaymentMethod: string(m.PaymentMethod),
		Status:        string(m.Status),
		PaidAt:        m.PaidAt,
		CallbackData:  m.CallbackData,
		Remark:        m.Remark,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
	if m.Edges.User != nil {
		order.User = userEntityToService(m.Edges.User)
	}
	if m.Edges.Product != nil {
		order.Product = productEntityToService(m.Edges.Product)
	}
	return order
}

func paymentOrderEntitiesToService(models []*dbent.PaymentOrder) []service.PaymentOrder {
	out := make([]service.PaymentOrder, 0, len(models))
	for _, m := range models {
		if o := paymentOrderEntityToService(m); o != nil {
			out = append(out, *o)
		}
	}
	return out
}

// deletableStatuses 可删除的订单状态
var deletableStatuses = []paymentorder.Status{
	paymentorder.StatusPending,
	paymentorder.StatusAuditing,
	paymentorder.StatusFailed,
}

func (r *paymentOrderRepository) Delete(ctx context.Context, id int64) error {
	// 先检查订单是否存在及状态
	order, err := r.client.PaymentOrder.Query().
		Where(paymentorder.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrPaymentOrderNotFound
		}
		return err
	}

	// 检查状态是否允许删除
	canDelete := false
	for _, s := range deletableStatuses {
		if order.Status == s {
			canDelete = true
			break
		}
	}
	if !canDelete {
		return service.ErrPaymentOrderCannotDelete
	}

	// 执行软删除（SoftDeleteMixin 会自动转换为 UPDATE deleted_at）
	return r.client.PaymentOrder.DeleteOneID(id).Exec(ctx)
}

func (r *paymentOrderRepository) BatchDelete(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	// 批量软删除符合条件的订单
	affected, err := r.client.PaymentOrder.Delete().
		Where(
			paymentorder.IDIn(ids...),
			paymentorder.StatusIn(deletableStatuses...),
		).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return affected, nil
}
