package handler

import (
	"fmt"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SocialOAuthHandler 社交登录 Handler
type SocialOAuthHandler struct {
	socialOAuthService *service.SocialOAuthService
}

// NewSocialOAuthHandler 创建社交登录 Handler
func NewSocialOAuthHandler(socialOAuthService *service.SocialOAuthService) *SocialOAuthHandler {
	return &SocialOAuthHandler{
		socialOAuthService: socialOAuthService,
	}
}

// GetProviders 获取启用的 OAuth 提供商列表
// GET /api/v1/social/providers
func (h *SocialOAuthHandler) GetProviders(c *gin.Context) {
	providers, err := h.socialOAuthService.GetEnabledProviders(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, providers)
}

// StartOAuthRequest 启动 OAuth 请求
type StartOAuthRequest struct {
	Provider   string `json:"provider" binding:"required"`
	RedirectTo string `json:"redirect_to"`
}

// StartOAuth 启动 OAuth 登录流程
// POST /api/v1/social/login/start
func (h *SocialOAuthHandler) StartOAuth(c *gin.Context) {
	var req StartOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 构建回调 URL
	callbackURL := h.buildCallbackURL(c, req.Provider, "login")

	result, err := h.socialOAuthService.StartOAuth(c.Request.Context(), req.Provider, req.RedirectTo, callbackURL)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// HandleCallbackRequest 处理回调请求
type HandleCallbackRequest struct {
	Provider  string `json:"provider" binding:"required"`
	Code      string `json:"code" binding:"required"`
	State     string `json:"state" binding:"required"`
	SessionID string `json:"session_id" binding:"required"`
}

// HandleCallback 处理 OAuth 回调
// POST /api/v1/social/login/callback
func (h *SocialOAuthHandler) HandleCallback(c *gin.Context) {
	var req HandleCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 构建回调 URL（需要与 StartOAuth 时一致）
	callbackURL := h.buildCallbackURL(c, req.Provider, "login")

	result, err := h.socialOAuthService.HandleCallback(c.Request.Context(), req.Provider, req.Code, req.State, req.SessionID, callbackURL)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// GetBindings 获取用户的第三方账号绑定列表
// GET /api/v1/social/bindings
func (h *SocialOAuthHandler) GetBindings(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		response.ErrorFrom(c, infraerrors.Unauthorized("UNAUTHORIZED", "user not authenticated"))
		return
	}

	bindings, err := h.socialOAuthService.GetUserBindings(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, bindings)
}

// StartBindRequest 启动绑定请求
type StartBindRequest struct {
	Provider string `json:"provider" binding:"required"`
}

// StartBind 启动账号绑定流程
// POST /api/v1/social/bind/start
func (h *SocialOAuthHandler) StartBind(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		response.ErrorFrom(c, infraerrors.Unauthorized("UNAUTHORIZED", "user not authenticated"))
		return
	}

	var req StartBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 构建回调 URL
	callbackURL := h.buildCallbackURL(c, req.Provider, "bind")

	result, err := h.socialOAuthService.StartBind(c.Request.Context(), userID, req.Provider, callbackURL)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// HandleBindCallbackRequest 处理绑定回调请求
type HandleBindCallbackRequest struct {
	Provider  string `json:"provider" binding:"required"`
	Code      string `json:"code" binding:"required"`
	State     string `json:"state" binding:"required"`
	SessionID string `json:"session_id" binding:"required"`
}

// HandleBindCallback 处理绑定回调
// POST /api/v1/social/bind/callback
func (h *SocialOAuthHandler) HandleBindCallback(c *gin.Context) {
	var req HandleBindCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 构建回调 URL
	callbackURL := h.buildCallbackURL(c, req.Provider, "bind")

	err := h.socialOAuthService.HandleBindCallback(c.Request.Context(), req.Provider, req.Code, req.State, req.SessionID, callbackURL)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Binding successful"})
}

// UnbindRequest 解绑请求
type UnbindRequest struct {
	Provider string `json:"provider" binding:"required"`
}

// Unbind 解绑第三方账号
// POST /api/v1/social/unbind
func (h *SocialOAuthHandler) Unbind(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		response.ErrorFrom(c, infraerrors.Unauthorized("UNAUTHORIZED", "user not authenticated"))
		return
	}

	var req UnbindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	err := h.socialOAuthService.Unbind(c.Request.Context(), userID, req.Provider)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Unbind successful"})
}

// buildCallbackURL 构建回调 URL
func (h *SocialOAuthHandler) buildCallbackURL(c *gin.Context, provider, action string) string {
	scheme := "https"
	if c.Request.TLS == nil && !strings.HasPrefix(c.Request.Host, "localhost") {
		// 检查 X-Forwarded-Proto
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if strings.HasPrefix(c.Request.Host, "localhost") || strings.HasPrefix(c.Request.Host, "127.0.0.1") {
			scheme = "http"
		}
	}

	// 前端回调页面路径 - 统一使用 /auth/social/{action}/callback
	callbackPath := fmt.Sprintf("/auth/social/%s/callback", action)

	return fmt.Sprintf("%s://%s%s?provider=%s", scheme, c.Request.Host, callbackPath, url.QueryEscape(provider))
}

// getUserIDFromContext 从上下文中获取用户 ID
func getUserIDFromContext(c *gin.Context) (int64, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := userID.(int64)
	return id, ok
}
