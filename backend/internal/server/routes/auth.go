package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/middleware"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterAuthRoutes 注册认证相关路由
func RegisterAuthRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth servermiddleware.JWTAuthMiddleware,
	redisClient *redis.Client,
) {
	// 创建速率限制器
	rateLimiter := middleware.NewRateLimiter(redisClient)

	// 公开接口
	auth := v1.Group("/auth")
	{
		auth.POST("/register", h.Auth.Register)
		auth.POST("/login", h.Auth.Login)
		auth.POST("/send-verify-code", h.Auth.SendVerifyCode)
		// 优惠码验证接口添加速率限制：每分钟最多 10 次（Redis 故障时 fail-close）
		auth.POST("/validate-promo-code", rateLimiter.LimitWithOptions("validate-promo", 10, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.ValidatePromoCode)
		auth.GET("/oauth/linuxdo/start", h.Auth.LinuxDoOAuthStart)
		auth.GET("/oauth/linuxdo/callback", h.Auth.LinuxDoOAuthCallback)
	}

	// 社交 OAuth 登录（无需认证）
	social := v1.Group("/social")
	{
		social.GET("/providers", h.SocialOAuth.GetProviders)
		social.POST("/login/start", h.SocialOAuth.StartOAuth)
		social.POST("/login/callback", h.SocialOAuth.HandleCallback)
		// 绑定回调也不需要认证，因为 userID 已经存储在 session 中
		// /bind/start 需要认证（在 user.go），回调时从 session 获取 userID
		social.POST("/bind/callback", h.SocialOAuth.HandleBindCallback)
	}

	// 公开设置（无需认证）
	settings := v1.Group("/settings")
	{
		settings.GET("/public", h.Setting.GetPublicSettings)
	}

	// 公告（无需认证）
	announcements := v1.Group("/announcements")
	{
		announcements.GET("/active", h.Announcement.GetActiveAnnouncements)
	}

	// 需要认证的当前用户信息
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	{
		authenticated.GET("/auth/me", h.Auth.GetCurrentUser)
	}
}
