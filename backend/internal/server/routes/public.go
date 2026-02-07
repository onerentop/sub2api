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

		// 支付回调接口 (YiPay异步通知，无需认证)
		payment := public.Group("/payment")
		{
			payment.POST("/callback", h.Payment.Callback)
			payment.GET("/callback", h.Payment.Callback) // 部分易支付使用 GET 回调
			payment.GET("/return", h.Payment.Return)     // 支付成功后同步跳转
		}
	}
}
