package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PaymentOrderHandler 管理端订单管理
type PaymentOrderHandler struct {
	paymentService *service.PaymentService
}

// NewPaymentOrderHandler 创建订单管理 Handler
func NewPaymentOrderHandler(paymentService *service.PaymentService) *PaymentOrderHandler {
	return &PaymentOrderHandler{
		paymentService: paymentService,
	}
}

// List 订单列表
// GET /api/v1/admin/payment-orders
func (h *PaymentOrderHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	status := c.Query("status")
	orderType := c.Query("order_type")
	paymentMethod := c.Query("payment_method")
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 100 {
		search = search[:100]
	}

	orders, paginationResult, err := h.paymentService.ListOrdersWithFilters(
		c.Request.Context(), params, status, orderType, paymentMethod, search,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Paginated(c, orders, paginationResult.Total, page, pageSize)
}

// GetByID 获取订单详情
// GET /api/v1/admin/payment-orders/:id
func (h *PaymentOrderHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	order, err := h.paymentService.GetOrderByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, order)
}

// GetStats 获取订单统计
// GET /api/v1/admin/payment-orders/stats
func (h *PaymentOrderHandler) GetStats(c *gin.Context) {
	stats, err := h.paymentService.GetOrderStats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, stats)
}

// ApproveRequest 审核通过请求
type ApproveRequest struct{}

// Approve 审核通过订单
// POST /api/v1/admin/payment-orders/:id/approve
func (h *PaymentOrderHandler) Approve(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	if err := h.paymentService.ApproveOrder(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Order approved successfully"})
}

// RejectRequest 审核拒绝请求
type RejectRequest struct {
	Reason string `json:"reason" binding:"required,max=500"`
}

// Reject 审核拒绝订单
// POST /api/v1/admin/payment-orders/:id/reject
func (h *PaymentOrderHandler) Reject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.paymentService.RejectOrder(c.Request.Context(), id, req.Reason); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Order rejected successfully"})
}

// ManualFulfillRequest 手动完成订单请求
type ManualFulfillRequest struct {
	TradeNo string `json:"trade_no" binding:"required,max=64"`
}

// ManualFulfill 手动完成订单（充值到账）
// POST /api/v1/admin/payment-orders/:id/fulfill
func (h *PaymentOrderHandler) ManualFulfill(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	var req ManualFulfillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.paymentService.ManualFulfillOrder(c.Request.Context(), id, req.TradeNo); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Order fulfilled successfully"})
}

// Delete 删除订单（仅允许 pending/auditing/failed 状态）
// DELETE /api/v1/admin/payment-orders/:id
func (h *PaymentOrderHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	if err := h.paymentService.DeleteOrder(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Order deleted successfully"})
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1,max=100"`
}

// BatchDelete 批量删除订单
// POST /api/v1/admin/payment-orders/batch-delete
func (h *PaymentOrderHandler) BatchDelete(c *gin.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	deleted, err := h.paymentService.BatchDeleteOrders(c.Request.Context(), req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "Orders deleted successfully",
		"deleted": deleted,
	})
}
