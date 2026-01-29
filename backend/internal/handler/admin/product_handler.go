package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ProductHandler 管理端商品管理
type ProductHandler struct {
	productRepo service.ProductRepository
}

// NewProductHandler 创建商品管理 Handler
func NewProductHandler(productRepo service.ProductRepository) *ProductHandler {
	return &ProductHandler{
		productRepo: productRepo,
	}
}

// CreateProductRequest 创建商品请求
type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required,max=100"`
	Description *string `json:"description"`
	Type        string  `json:"type" binding:"required,oneof=balance subscription"`
	PriceCNY    float64 `json:"price_cny" binding:"required,gt=0"`
	Value       float64 `json:"value" binding:"required,gt=0"`
	GroupID     *int64  `json:"group_id"` // 订阅类型必填
	IsActive    bool    `json:"is_active"`
	SortOrder   int     `json:"sort_order"`
}

// UpdateProductRequest 更新商品请求
type UpdateProductRequest struct {
	Name        string  `json:"name" binding:"required,max=100"`
	Description *string `json:"description"`
	Type        *string `json:"type" binding:"omitempty,oneof=balance subscription"` // 可选，不传则保持原类型
	PriceCNY    float64 `json:"price_cny" binding:"required,gt=0"`
	Value       float64 `json:"value" binding:"required,gt=0"`
	GroupID     *int64  `json:"group_id"`
	IsActive    bool    `json:"is_active"`
	SortOrder   int     `json:"sort_order"`
}

// List 商品列表
// GET /api/v1/admin/products
func (h *ProductHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	productType := c.Query("type")
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 100 {
		search = search[:100]
	}

	var isActive *bool
	if activeStr := c.Query("is_active"); activeStr != "" {
		active := activeStr == "true" || activeStr == "1"
		isActive = &active
	}

	products, paginationResult, err := h.productRepo.ListWithFilters(c.Request.Context(), params, productType, isActive, search)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Paginated(c, products, paginationResult.Total, page, pageSize)
}

// GetByID 获取商品详情
// GET /api/v1/admin/products/:id
func (h *ProductHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid product ID")
		return
	}

	product, err := h.productRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, product)
}

// Create 创建商品
// POST /api/v1/admin/products
func (h *ProductHandler) Create(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 订阅类型必须关联分组
	if req.Type == service.ProductTypeSubscription && req.GroupID == nil {
		response.BadRequest(c, "Subscription product must have group_id")
		return
	}

	product := &service.Product{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		PriceCNY:    req.PriceCNY,
		Value:       req.Value,
		GroupID:     req.GroupID,
		IsActive:    req.IsActive,
		SortOrder:   req.SortOrder,
	}

	if err := h.productRepo.Create(c.Request.Context(), product); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, product)
}

// Update 更新商品
// PUT /api/v1/admin/products/:id
func (h *ProductHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid product ID")
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 获取现有商品
	product, err := h.productRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 确定最终类型（如果传了就用新的，否则保持原类型）
	finalType := product.Type
	if req.Type != nil {
		finalType = *req.Type
	}

	// 订阅类型必须关联分组
	if finalType == service.ProductTypeSubscription && req.GroupID == nil {
		response.BadRequest(c, "Subscription product must have group_id")
		return
	}

	// 更新字段
	product.Name = req.Name
	product.Description = req.Description
	product.Type = finalType
	product.PriceCNY = req.PriceCNY
	product.Value = req.Value
	product.GroupID = req.GroupID
	product.IsActive = req.IsActive
	product.SortOrder = req.SortOrder

	if err := h.productRepo.Update(c.Request.Context(), product); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, product)
}

// Delete 删除商品（软删除）
// DELETE /api/v1/admin/products/:id
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid product ID")
		return
	}

	if err := h.productRepo.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Product deleted successfully"})
}

// ToggleActive 切换商品上下架状态
// POST /api/v1/admin/products/:id/toggle-active
func (h *ProductHandler) ToggleActive(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid product ID")
		return
	}

	product, err := h.productRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 切换状态
	product.IsActive = !product.IsActive

	if err := h.productRepo.Update(c.Request.Context(), product); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, product)
}
