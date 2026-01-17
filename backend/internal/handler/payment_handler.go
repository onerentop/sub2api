package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PaymentHandler 处理用户端支付相关请求
type PaymentHandler struct {
	paymentService *service.PaymentService
	productRepo    service.ProductRepository
	config         *config.PaymentConfig
}

// NewPaymentHandler 创建支付 Handler
func NewPaymentHandler(
	paymentService *service.PaymentService,
	productRepo service.ProductRepository,
	cfg *config.Config,
) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		productRepo:    productRepo,
		config:         &cfg.Payment,
	}
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	ProductID     *int64  `json:"product_id"`
	CustomAmount  float64 `json:"custom_amount"`
	PaymentMethod string  `json:"payment_method" binding:"required,oneof=alipay wechat"`
}

// GetProducts 获取商品列表（上架的商品）
// GET /api/v1/payment/products
func (h *PaymentHandler) GetProducts(c *gin.Context) {
	if !h.config.Enabled {
		response.BadRequest(c, "Payment is not enabled")
		return
	}

	products, err := h.productRepo.ListActive(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, products)
}

// CreateOrder 创建支付订单
// POST /api/v1/payment/orders
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	if !h.config.Enabled {
		response.BadRequest(c, "Payment is not enabled")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 至少需要 ProductID 或 CustomAmount
	if req.ProductID == nil && req.CustomAmount <= 0 {
		response.BadRequest(c, "Either product_id or custom_amount is required")
		return
	}

	svcReq := &service.CreateOrderRequest{
		UserID:        subject.UserID,
		ProductID:     req.ProductID,
		CustomAmount:  req.CustomAmount,
		PaymentMethod: req.PaymentMethod,
	}

	result, err := h.paymentService.CreateOrder(c.Request.Context(), svcReq)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// GetOrderStatus 查询订单状态
// GET /api/v1/payment/orders/:order_no/status
func (h *PaymentHandler) GetOrderStatus(c *gin.Context) {
	if !h.config.Enabled {
		response.BadRequest(c, "Payment is not enabled")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	orderNo := c.Param("order_no")
	if orderNo == "" {
		response.BadRequest(c, "Order number is required")
		return
	}

	order, err := h.paymentService.QueryOrderStatus(c.Request.Context(), orderNo)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 验证订单属于当前用户
	if order.UserID != subject.UserID {
		response.Forbidden(c, "Order not found")
		return
	}

	response.Success(c, gin.H{
		"order_no":  order.OrderNo,
		"status":    order.Status,
		"amount":    order.AmountCNY,
		"paid_at":   order.PaidAt,
	})
}

// ListOrders 获取用户订单列表
// GET /api/v1/payment/orders
func (h *PaymentHandler) ListOrders(c *gin.Context) {
	if !h.config.Enabled {
		response.BadRequest(c, "Payment is not enabled")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	orders, paginationResult, err := h.paymentService.ListUserOrders(c.Request.Context(), subject.UserID, params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Paginated(c, orders, paginationResult.Total, page, pageSize)
}

// Callback 处理易支付回调
// POST /api/v1/payment/callback
func (h *PaymentHandler) Callback(c *gin.Context) {
	if !h.config.Enabled {
		c.String(200, "fail")
		return
	}

	// 获取所有参数（GET 和 POST）
	data := make(map[string]string)

	// 尝试从 query 获取
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			data[k] = v[0]
		}
	}

	// 尝试从 form 获取（POST）
	if err := c.Request.ParseForm(); err == nil {
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				data[k] = v[0]
			}
		}
	}

	if err := h.paymentService.HandleCallback(c.Request.Context(), data); err != nil {
		// 回调失败，返回 fail 让易支付重试
		c.String(200, "fail")
		return
	}

	// 回调成功，返回 success
	c.String(200, "success")
}

// Return 处理支付成功后的同步跳转
// GET /api/v1/payment/return
func (h *PaymentHandler) Return(c *gin.Context) {
	orderNo := c.Query("out_trade_no")

	// 跳转到前端充值页面，带上订单号
	redirectURL := "/recharge"
	if orderNo != "" {
		redirectURL += "?order_no=" + orderNo
	}

	c.Redirect(302, redirectURL)
}

// GetPaymentConfig 获取支付配置（公开信息）
// GET /api/v1/payment/config
func (h *PaymentHandler) GetPaymentConfig(c *gin.Context) {
	response.Success(c, gin.H{
		"enabled":         h.config.Enabled,
		"min_amount":      h.config.MinAmount,
		"max_amount":      h.config.MaxAmount,
		"audit_threshold": h.config.AuditThreshold,
		"payment_methods": []string{"alipay", "wechat"},
		"cny_usd_rate":    h.config.CNYToValueRate,
	})
}
