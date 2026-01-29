package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// PaymentService 支付服务
// 负责支付业务的核心逻辑：创建订单、处理回调、充值到账
type PaymentService struct {
	config              *config.PaymentConfig
	orderRepo           PaymentOrderRepository
	productRepo         ProductRepository
	userRepo            UserRepository
	subscriptionService *SubscriptionService
	yipayService        *YiPayService
	entClient           *dbent.Client
	billingCacheService *BillingCacheService
	settingService      *SettingService
}

// NewPaymentService 创建支付服务实例
func NewPaymentService(
	cfg *config.Config,
	orderRepo PaymentOrderRepository,
	productRepo ProductRepository,
	userRepo UserRepository,
	subscriptionService *SubscriptionService,
	yipayService *YiPayService,
	entClient *dbent.Client,
	billingCacheService *BillingCacheService,
) *PaymentService {
	return &PaymentService{
		config:              &cfg.Payment,
		orderRepo:           orderRepo,
		productRepo:         productRepo,
		userRepo:            userRepo,
		subscriptionService: subscriptionService,
		yipayService:        yipayService,
		entClient:           entClient,
		billingCacheService: billingCacheService,
	}
}

// SetSettingService 注入 SettingService 以支持动态配置
func (s *PaymentService) SetSettingService(settingService *SettingService) {
	s.settingService = settingService
}

// getConfig 获取支付配置（优先动态配置，回退静态配置）
func (s *PaymentService) getConfig(ctx context.Context) *config.PaymentConfig {
	if s.settingService != nil {
		cfg, err := s.settingService.GetPaymentConfig(ctx)
		if err == nil && cfg != nil {
			return cfg
		}
	}
	return s.config
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	UserID        int64
	ProductID     *int64  // 商品 ID（可选）
	CustomAmount  float64 // 自定义金额（可选，与 ProductID 二选一）
	PaymentMethod string  // 支付方式：alipay/wechat
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	OrderNo     string  `json:"order_no"`     // 订单号
	PaymentURL  string  `json:"payment_url"`  // 支付链接
	AmountCNY   float64 `json:"amount_cny"`   // 支付金额（人民币）
	AmountValue float64 `json:"amount_value"` // 余额值
	OrderType   string  `json:"order_type"`   // 订单类型
}

// CreateOrder 创建支付订单
func (s *PaymentService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	cfg := s.getConfig(ctx)
	if !cfg.Enabled {
		return nil, fmt.Errorf("payment is not enabled")
	}

	var (
		amountCNY   float64
		amountValue float64
		orderType   = ProductTypeBalance
		productName = "余额充值"
		productID   *int64
		groupID     *int64
	)

	// 确定订单金额和类型
	if req.ProductID != nil {
		// 使用商品
		product, err := s.productRepo.GetByID(ctx, *req.ProductID)
		if err != nil {
			return nil, err
		}
		if !product.IsActive {
			return nil, ErrProductNotFound
		}
		amountCNY = product.PriceCNY
		amountValue = product.Value
		orderType = product.Type
		productID = &product.ID
		groupID = product.GroupID

		if product.Type == ProductTypeSubscription {
			productName = fmt.Sprintf("订阅套餐 - %s", product.Name)
		} else {
			productName = fmt.Sprintf("余额充值 - %s", product.Name)
		}
	} else if req.CustomAmount > 0 {
		// 自定义金额充值
		if req.CustomAmount < cfg.MinAmount {
			return nil, fmt.Errorf("minimum amount is %.2f", cfg.MinAmount)
		}
		if req.CustomAmount > cfg.MaxAmount {
			return nil, fmt.Errorf("maximum amount is %.2f", cfg.MaxAmount)
		}
		amountCNY = req.CustomAmount
		amountValue = req.CustomAmount * cfg.CNYToValueRate
		productName = fmt.Sprintf("余额充值 ¥%.2f", req.CustomAmount)
	} else {
		return nil, ErrPaymentAmountInvalid
	}

	// 生成订单号
	orderNo, err := generateOrderNo()
	if err != nil {
		return nil, fmt.Errorf("generate order no: %w", err)
	}

	// 确定订单状态（大额订单需要审核）
	status := PaymentOrderStatusPending
	if amountCNY >= cfg.AuditThreshold {
		status = PaymentOrderStatusAuditing
	}

	// 创建订单
	order := &PaymentOrder{
		UserID:        req.UserID,
		ProductID:     productID,
		OrderNo:       orderNo,
		AmountCNY:     amountCNY,
		AmountValue:   amountValue,
		OrderType:     orderType,
		PaymentMethod: req.PaymentMethod,
		Status:        status,
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// 如果需要审核，直接返回（不生成支付链接）
	if status == PaymentOrderStatusAuditing {
		return &CreateOrderResponse{
			OrderNo:     orderNo,
			AmountCNY:   amountCNY,
			AmountValue: amountValue,
			OrderType:   orderType,
		}, nil
	}

	// 生成支付链接
	paymentType := PaymentTypeAlipay
	if req.PaymentMethod == PaymentMethodWeChat {
		paymentType = PaymentTypeWechat
	}

	payResp, err := s.yipayService.CreatePayment(ctx, &CreatePaymentRequest{
		OrderNo:     orderNo,
		Amount:      amountCNY,
		ProductName: productName,
		PaymentType: paymentType,
	})
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	// 如果是订阅类型，保存 groupID 到订单备注
	if orderType == ProductTypeSubscription && groupID != nil {
		remark := fmt.Sprintf("group_id:%d", *groupID)
		order.Remark = &remark
		if err := s.orderRepo.Update(ctx, order); err != nil {
			// 非关键错误，只记录日志
			fmt.Printf("update order remark failed: %v\n", err)
		}
	}

	return &CreateOrderResponse{
		OrderNo:     orderNo,
		PaymentURL:  payResp.PaymentURL,
		AmountCNY:   amountCNY,
		AmountValue: amountValue,
		OrderType:   orderType,
	}, nil
}

// HandleCallback 处理支付回调
func (s *PaymentService) HandleCallback(ctx context.Context, data map[string]string) error {
	// 验证签名
	callbackData, err := s.yipayService.VerifyCallback(data)
	if err != nil {
		return err
	}

	// 只处理成功的交易
	if !callbackData.IsTradeSuccess() {
		return nil
	}

	// 获取订单
	order, err := s.orderRepo.GetByOrderNo(ctx, callbackData.OutTradeNo)
	if err != nil {
		return err
	}

	// 检查订单状态
	if order.IsPaid() {
		// 已支付，重复回调，直接返回成功
		return nil
	}
	if !order.CanPay() {
		return ErrPaymentOrderPending
	}

	// 验证金额
	if order.AmountCNY != callbackData.Money {
		return fmt.Errorf("amount mismatch: expected %.2f, got %.2f", order.AmountCNY, callbackData.Money)
	}

	// 标记订单为已支付
	callbackJSON := make(map[string]any)
	for k, v := range callbackData.RawData {
		callbackJSON[k] = v
	}

	if err := s.orderRepo.MarkAsPaid(ctx, order.ID, callbackData.TradeNo, callbackJSON); err != nil {
		return err
	}

	// 执行充值到账
	return s.processOrderFulfillment(ctx, order)
}

// processOrderFulfillment 处理订单履约（充值到账）
func (s *PaymentService) processOrderFulfillment(ctx context.Context, order *PaymentOrder) error {
	switch order.OrderType {
	case ProductTypeBalance:
		// 余额充值
		return s.addUserBalance(ctx, order.UserID, order.AmountValue)
	case ProductTypeSubscription:
		// 订阅充值
		return s.addUserSubscription(ctx, order)
	default:
		return fmt.Errorf("unknown order type: %s", order.OrderType)
	}
}

// addUserBalance 增加用户余额
// 注意：UpdateBalance 内部使用 AddBalance 原子操作，直接传入增量即可
func (s *PaymentService) addUserBalance(ctx context.Context, userID int64, amount float64) error {
	if err := s.userRepo.UpdateBalance(ctx, userID, amount); err != nil {
		return err
	}

	// 清除计费缓存
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateUserBalance(ctx, userID)
	}

	return nil
}

// addUserSubscription 增加用户订阅
func (s *PaymentService) addUserSubscription(ctx context.Context, order *PaymentOrder) error {
	// 从产品获取分组 ID 和天数
	if order.ProductID == nil {
		return fmt.Errorf("subscription order must have product_id")
	}

	product, err := s.productRepo.GetByID(ctx, *order.ProductID)
	if err != nil {
		return err
	}

	if product.GroupID == nil {
		return fmt.Errorf("subscription product must have group_id")
	}

	// 计算订阅天数（value 字段存储天数）
	days := int(product.Value)
	if days <= 0 {
		days = 30 // 默认 30 天
	}

	// 创建或延长订阅
	input := &AssignSubscriptionInput{
		UserID:       order.UserID,
		GroupID:      *product.GroupID,
		ValidityDays: days,
		AssignedBy:   0, // 系统自动分配
		Notes:        fmt.Sprintf("Payment order: %s", order.OrderNo),
	}
	_, _, err = s.subscriptionService.AssignOrExtendSubscription(ctx, input)
	return err
}

// GetOrder 获取订单详情
func (s *PaymentService) GetOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	return s.orderRepo.GetByOrderNo(ctx, orderNo)
}

// GetOrderByID 通过 ID 获取订单
func (s *PaymentService) GetOrderByID(ctx context.Context, id int64) (*PaymentOrder, error) {
	return s.orderRepo.GetByID(ctx, id)
}

// ListUserOrders 获取用户订单列表
func (s *PaymentService) ListUserOrders(ctx context.Context, userID int64, params pagination.PaginationParams) ([]PaymentOrder, *pagination.PaginationResult, error) {
	return s.orderRepo.ListByUser(ctx, userID, params)
}

// QueryOrderStatus 查询订单支付状态（轮询用）
func (s *PaymentService) QueryOrderStatus(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	order, err := s.orderRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}

	// 如果订单已支付或非 pending 状态，直接返回
	if !order.IsPending() {
		return order, nil
	}

	// 尝试主动查询易支付状态
	callbackData, err := s.yipayService.QueryOrder(ctx, orderNo)
	if err != nil {
		// 查询失败不影响返回，可能是易支付不支持查询
		return order, nil
	}

	// 如果查询到已支付
	if callbackData.IsTradeSuccess() {
		// 标记订单为已支付
		callbackJSON := map[string]any{
			"trade_no":     callbackData.TradeNo,
			"trade_status": callbackData.TradeStatus,
			"query_time":   time.Now().Format(time.RFC3339),
		}

		if err := s.orderRepo.MarkAsPaid(ctx, order.ID, callbackData.TradeNo, callbackJSON); err != nil {
			return nil, err
		}

		// 执行充值到账
		if err := s.processOrderFulfillment(ctx, order); err != nil {
			return nil, err
		}

		// 重新获取订单
		return s.orderRepo.GetByOrderNo(ctx, orderNo)
	}

	return order, nil
}

// ApproveOrder 审核通过订单（管理员操作）
func (s *PaymentService) ApproveOrder(ctx context.Context, orderID int64) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	if !order.IsAuditing() {
		return fmt.Errorf("order is not in auditing status")
	}

	// 更新状态为 pending，等待支付
	return s.orderRepo.UpdateStatus(ctx, orderID, PaymentOrderStatusPending)
}

// RejectOrder 审核拒绝订单（管理员操作）
func (s *PaymentService) RejectOrder(ctx context.Context, orderID int64, reason string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	if !order.IsAuditing() {
		return fmt.Errorf("order is not in auditing status")
	}

	// 更新备注和状态
	order.Remark = &reason
	order.Status = PaymentOrderStatusFailed
	return s.orderRepo.Update(ctx, order)
}

// ManualFulfillOrder 手动完成订单（管理员操作，用于异常情况）
func (s *PaymentService) ManualFulfillOrder(ctx context.Context, orderID int64, tradeNo string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	if order.IsPaid() {
		return errors.New("order already paid")
	}

	// 标记为已支付
	callbackJSON := map[string]any{
		"manual":      true,
		"fulfill_at":  time.Now().Format(time.RFC3339),
		"admin_note":  "Manual fulfillment",
	}

	if err := s.orderRepo.MarkAsPaid(ctx, order.ID, tradeNo, callbackJSON); err != nil {
		return err
	}

	// 执行充值到账
	return s.processOrderFulfillment(ctx, order)
}

// GetOrderStats 获取订单统计
func (s *PaymentService) GetOrderStats(ctx context.Context) (*PaymentOrderStats, error) {
	return s.orderRepo.GetOrderStats(ctx)
}

// ListOrdersWithFilters 获取订单列表（管理员用，带过滤）
func (s *PaymentService) ListOrdersWithFilters(ctx context.Context, params pagination.PaginationParams, status, orderType, paymentMethod, search string) ([]PaymentOrder, *pagination.PaginationResult, error) {
	return s.orderRepo.ListWithFilters(ctx, params, status, orderType, paymentMethod, search)
}

// generateOrderNo 生成订单号
func generateOrderNo() (string, error) {
	// 格式：日期 + 随机字符串
	datePrefix := time.Now().Format("20060102150405")

	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	randomSuffix := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("%s%s", datePrefix, randomSuffix), nil
}
