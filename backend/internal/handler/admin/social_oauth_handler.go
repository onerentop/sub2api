package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SocialOAuthHandler 社交登录管理 Handler
type SocialOAuthHandler struct {
	socialOAuthService *service.SocialOAuthService
}

// NewSocialOAuthHandler 创建社交登录管理 Handler
func NewSocialOAuthHandler(socialOAuthService *service.SocialOAuthService) *SocialOAuthHandler {
	return &SocialOAuthHandler{
		socialOAuthService: socialOAuthService,
	}
}

// ListProviders 获取所有 OAuth 提供商配置
// GET /api/v1/admin/social-oauth/providers
func (h *SocialOAuthHandler) ListProviders(c *gin.Context) {
	providers, err := h.socialOAuthService.GetAllProviders(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 转换为响应格式（隐藏敏感信息）
	result := make([]OAuthProviderResponse, 0, len(providers))
	for _, p := range providers {
		result = append(result, OAuthProviderResponse{
			Name:            p.Name,
			DisplayName:     p.DisplayName,
			Enabled:         p.Enabled,
			HasClientID:     p.ClientID != "",
			HasClientSecret: p.ClientSecret != "",
			Config:          p.Config,
			CreatedAt:       p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:       p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	response.Success(c, result)
}

// OAuthProviderResponse OAuth 提供商响应
type OAuthProviderResponse struct {
	Name            string         `json:"name"`
	DisplayName     string         `json:"display_name"`
	Enabled         bool           `json:"enabled"`
	HasClientID     bool           `json:"has_client_id"`
	HasClientSecret bool           `json:"has_client_secret"`
	Config          map[string]any `json:"config,omitempty"`
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
}

// UpdateProviderRequest 更新提供商请求
type UpdateProviderRequest struct {
	ClientID     *string        `json:"client_id"`
	ClientSecret *string        `json:"client_secret"`
	Enabled      *bool          `json:"enabled"`
	Config       map[string]any `json:"config,omitempty"`
}

// UpdateProvider 更新 OAuth 提供商配置
// PUT /api/v1/admin/social-oauth/providers/:name
func (h *SocialOAuthHandler) UpdateProvider(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		response.BadRequest(c, "Provider name is required")
		return
	}

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := &service.UpdateOAuthProviderInput{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		Enabled:      req.Enabled,
		Config:       req.Config,
	}

	provider, err := h.socialOAuthService.UpdateProvider(c.Request.Context(), name, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, OAuthProviderResponse{
		Name:            provider.Name,
		DisplayName:     provider.DisplayName,
		Enabled:         provider.Enabled,
		HasClientID:     provider.ClientID != "",
		HasClientSecret: provider.ClientSecret != "",
		Config:          provider.Config,
		CreatedAt:       provider.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       provider.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// GetProvider 获取单个 OAuth 提供商配置（包含完整信息）
// GET /api/v1/admin/social-oauth/providers/:name
func (h *SocialOAuthHandler) GetProvider(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		response.BadRequest(c, "Provider name is required")
		return
	}

	providers, err := h.socialOAuthService.GetAllProviders(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	for _, p := range providers {
		if p.Name == name {
			response.Success(c, OAuthProviderDetailResponse{
				Name:         p.Name,
				DisplayName:  p.DisplayName,
				ClientID:     p.ClientID,
				Enabled:      p.Enabled,
				Config:       p.Config,
				CreatedAt:    p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:    p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
			return
		}
	}

	response.ErrorFrom(c, service.ErrOAuthProviderNotFound)
}

// OAuthProviderDetailResponse OAuth 提供商详细响应（包含 ClientID）
type OAuthProviderDetailResponse struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	ClientID    string         `json:"client_id"`
	Enabled     bool           `json:"enabled"`
	Config      map[string]any `json:"config,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}
