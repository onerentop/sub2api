package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// PublicAntigravityOAuthHandler 处理公开的 Antigravity OAuth 请求
type PublicAntigravityOAuthHandler struct {
	publicAccountService *service.PublicAccountService
}

// NewPublicAntigravityOAuthHandler 创建公开 OAuth handler 实例
func NewPublicAntigravityOAuthHandler(publicAccountService *service.PublicAccountService) *PublicAntigravityOAuthHandler {
	return &PublicAntigravityOAuthHandler{
		publicAccountService: publicAccountService,
	}
}

// Start 生成 OAuth 授权链接
// POST /public/antigravity/oauth/start
func (h *PublicAntigravityOAuthHandler) Start(c *gin.Context) {
	result, err := h.publicAccountService.GenerateAuthURL(c.Request.Context())
	if err != nil {
		response.InternalError(c, "生成授权链接失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// CompleteRequest 完成 OAuth 请求体
type CompleteRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	State     string `json:"state" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

// Complete 完成 OAuth 流程，创建或更新账户
// POST /public/antigravity/oauth/complete
func (h *PublicAntigravityOAuthHandler) Complete(c *gin.Context) {
	var req CompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数无效: "+err.Error())
		return
	}

	result, err := h.publicAccountService.CompleteOAuth(c.Request.Context(), &service.PublicOAuthCompleteInput{
		SessionID: req.SessionID,
		State:     req.State,
		Code:      req.Code,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, result)
}

// WakeRequest 唤醒请求体
type WakeRequest struct {
	SessionID string   `json:"session_id" binding:"required"`
	Models    []string `json:"models,omitempty"`
}

// Wake 执行唤醒请求（发送测试消息触发配额）
// POST /public/antigravity/wake
func (h *PublicAntigravityOAuthHandler) Wake(c *gin.Context) {
	var req WakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数无效: "+err.Error())
		return
	}

	result, err := h.publicAccountService.Wake(c.Request.Context(), &service.PublicWakeInput{
		SessionID: req.SessionID,
		Models:    req.Models,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, result)
}
