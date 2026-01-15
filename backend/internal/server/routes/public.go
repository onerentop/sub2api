package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes 注册公开路由（无需认证）
func RegisterPublicRoutes(router *gin.Engine, h *handler.Handlers) {
	public := router.Group("/public")
	{
		// Antigravity OAuth 公开接口
		antigravity := public.Group("/antigravity")
		{
			antigravity.POST("/oauth/start", h.PublicAntigravityOAuth.Start)
			antigravity.POST("/oauth/complete", h.PublicAntigravityOAuth.Complete)
			antigravity.POST("/wake", h.PublicAntigravityOAuth.Wake)
		}
	}
}
