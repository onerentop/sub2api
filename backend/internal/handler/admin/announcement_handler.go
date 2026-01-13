package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AnnouncementHandler handles admin announcement management
type AnnouncementHandler struct {
	announcementService *service.AnnouncementService
}

// NewAnnouncementHandler creates a new admin announcement handler
func NewAnnouncementHandler(announcementService *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementService: announcementService,
	}
}

// CreateAnnouncementRequest represents create announcement request
type CreateAnnouncementRequest struct {
	Title     string `json:"title"`                      // 可选
	Content   string `json:"content" binding:"required"` // 必填
	Type      string `json:"type" binding:"omitempty,oneof=info success warning error"`
	Enabled   *bool  `json:"enabled"`    // 默认 true
	StartTime *int64 `json:"start_time"` // 生效时间戳（秒）
	EndTime   *int64 `json:"end_time"`   // 过期时间戳（秒）
}

// UpdateAnnouncementRequest represents update announcement request
type UpdateAnnouncementRequest struct {
	Title     *string `json:"title"`
	Content   *string `json:"content"`
	Type      *string `json:"type" binding:"omitempty,oneof=info success warning error"`
	SortOrder *int    `json:"sort_order"`
	Enabled   *bool   `json:"enabled"`
	StartTime *int64  `json:"start_time"`
	EndTime   *int64  `json:"end_time"`
	// 用于清除时间字段（设为 0）
	ClearStartTime bool `json:"clear_start_time"`
	ClearEndTime   bool `json:"clear_end_time"`
}

// UpdateSortOrdersRequest represents batch sort order update request
type UpdateSortOrdersRequest struct {
	Orders []SortOrderItem `json:"orders" binding:"required,min=1"`
}

// SortOrderItem represents a single sort order item
type SortOrderItem struct {
	ID        int64 `json:"id" binding:"required"`
	SortOrder int   `json:"sort_order"`
}

// List handles listing all announcements with pagination
// GET /api/admin/announcements
func (h *AnnouncementHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)

	// Parse enabled filter
	var enabled *bool
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		switch enabledStr {
		case "true":
			b := true
			enabled = &b
		case "false":
			b := false
			enabled = &b
		}
	}

	announcementType := c.Query("type")

	params := pagination.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}

	announcements, paginationResult, err := h.announcementService.List(c.Request.Context(), params, enabled, announcementType)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.Announcement, 0, len(announcements))
	for i := range announcements {
		out = append(out, *dto.AnnouncementFromService(&announcements[i]))
	}
	response.Paginated(c, out, paginationResult.Total, page, pageSize)
}

// GetByID handles getting an announcement by ID
// GET /api/admin/announcements/:id
func (h *AnnouncementHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	announcement, err := h.announcementService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AnnouncementFromService(announcement))
}

// Create handles creating a new announcement
// POST /api/admin/announcements
func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := &service.CreateAnnouncementInput{
		Title:   req.Title,
		Content: req.Content,
		Type:    service.AnnouncementType(req.Type),
		Enabled: true,
	}

	if req.Enabled != nil {
		input.Enabled = *req.Enabled
	}

	if input.Type == "" {
		input.Type = service.AnnouncementTypeInfo
	}

	if req.StartTime != nil && *req.StartTime > 0 {
		t := time.Unix(*req.StartTime, 0)
		input.StartTime = &t
	}

	if req.EndTime != nil && *req.EndTime > 0 {
		t := time.Unix(*req.EndTime, 0)
		input.EndTime = &t
	}

	announcement, err := h.announcementService.Create(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AnnouncementFromService(announcement))
}

// Update handles updating an announcement
// PUT /api/admin/announcements/:id
func (h *AnnouncementHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	var req UpdateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := &service.UpdateAnnouncementInput{
		Title:          req.Title,
		Content:        req.Content,
		SortOrder:      req.SortOrder,
		Enabled:        req.Enabled,
		ClearStartTime: req.ClearStartTime,
		ClearEndTime:   req.ClearEndTime,
	}

	if req.Type != nil {
		t := service.AnnouncementType(*req.Type)
		input.Type = &t
	}

	if req.StartTime != nil {
		if *req.StartTime == 0 {
			input.ClearStartTime = true
		} else {
			t := time.Unix(*req.StartTime, 0)
			input.StartTime = &t
		}
	}

	if req.EndTime != nil {
		if *req.EndTime == 0 {
			input.ClearEndTime = true
		} else {
			t := time.Unix(*req.EndTime, 0)
			input.EndTime = &t
		}
	}

	announcement, err := h.announcementService.Update(c.Request.Context(), id, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AnnouncementFromService(announcement))
}

// Delete handles deleting an announcement
// DELETE /api/admin/announcements/:id
func (h *AnnouncementHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	err = h.announcementService.Delete(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Announcement deleted successfully"})
}

// UpdateSortOrders handles batch updating sort orders
// PUT /api/admin/announcements/sort
func (h *AnnouncementHandler) UpdateSortOrders(c *gin.Context) {
	var req UpdateSortOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	items := make([]service.AnnouncementSortItem, 0, len(req.Orders))
	for _, o := range req.Orders {
		items = append(items, service.AnnouncementSortItem{
			ID:        o.ID,
			SortOrder: o.SortOrder,
		})
	}

	if err := h.announcementService.UpdateSortOrders(c.Request.Context(), items); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Sort orders updated successfully"})
}
