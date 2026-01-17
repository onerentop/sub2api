package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
)

// AdminHandlers contains all admin-related HTTP handlers
type AdminHandlers struct {
	Dashboard        *admin.DashboardHandler
	User             *admin.UserHandler
	Group            *admin.GroupHandler
	Account          *admin.AccountHandler
	OAuth            *admin.OAuthHandler
	OpenAIOAuth      *admin.OpenAIOAuthHandler
	GeminiOAuth      *admin.GeminiOAuthHandler
	AntigravityOAuth *admin.AntigravityOAuthHandler
	SocialOAuth      *admin.SocialOAuthHandler
	Proxy            *admin.ProxyHandler
	Redeem           *admin.RedeemHandler
	Promo            *admin.PromoHandler
	Announcement     *admin.AnnouncementHandler
	Setting          *admin.SettingHandler
	Ops              *admin.OpsHandler
	System           *admin.SystemHandler
	Subscription     *admin.SubscriptionHandler
	Usage            *admin.UsageHandler
	UserAttribute    *admin.UserAttributeHandler
	Product          *admin.ProductHandler
	PaymentOrder     *admin.PaymentOrderHandler
}

// Handlers contains all HTTP handlers
type Handlers struct {
	Auth                   *AuthHandler
	User                   *UserHandler
	APIKey                 *APIKeyHandler
	Usage                  *UsageHandler
	Redeem                 *RedeemHandler
	Subscription           *SubscriptionHandler
	Announcement           *AnnouncementHandler
	Admin                  *AdminHandlers
	Gateway                *GatewayHandler
	OpenAIGateway          *OpenAIGatewayHandler
	Setting                *SettingHandler
	PublicAntigravityOAuth *PublicAntigravityOAuthHandler
	SocialOAuth            *SocialOAuthHandler
	Payment                *PaymentHandler
}

// BuildInfo contains build-time information
type BuildInfo struct {
	Version   string
	BuildType string // "source" for manual builds, "release" for CI builds
}
