package repository

import (
	"database/sql"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

// ProvideConcurrencyCache 创建并发控制缓存，从配置读取 TTL 参数
// 性能优化：TTL 可配置，支持长时间运行的 LLM 请求场景
func ProvideConcurrencyCache(rdb *redis.Client, cfg *config.Config) service.ConcurrencyCache {
	waitTTLSeconds := int(cfg.Gateway.Scheduling.StickySessionWaitTimeout.Seconds())
	if cfg.Gateway.Scheduling.FallbackWaitTimeout > cfg.Gateway.Scheduling.StickySessionWaitTimeout {
		waitTTLSeconds = int(cfg.Gateway.Scheduling.FallbackWaitTimeout.Seconds())
	}
	if waitTTLSeconds <= 0 {
		waitTTLSeconds = cfg.Gateway.ConcurrencySlotTTLMinutes * 60
	}
	return NewConcurrencyCache(rdb, cfg.Gateway.ConcurrencySlotTTLMinutes, waitTTLSeconds)
}

// ProvideGitHubReleaseClient 创建 GitHub Release 客户端
// 从配置中读取代理设置，支持国内服务器通过代理访问 GitHub
func ProvideGitHubReleaseClient(cfg *config.Config) service.GitHubReleaseClient {
	return NewGitHubReleaseClient(cfg.Update.ProxyURL)
}

// ProvidePricingRemoteClient 创建定价数据远程客户端
// 从配置中读取代理设置，支持国内服务器通过代理访问 GitHub 上的定价数据
func ProvidePricingRemoteClient(cfg *config.Config) service.PricingRemoteClient {
	return NewPricingRemoteClient(cfg.Update.ProxyURL)
}

// ProvideSessionLimitCache 创建会话限制缓存
// 用于 Anthropic OAuth/SetupToken 账号的并发会话数量控制
func ProvideSessionLimitCache(rdb *redis.Client, cfg *config.Config) service.SessionLimitCache {
	defaultIdleTimeoutMinutes := 5 // 默认 5 分钟空闲超时
	if cfg != nil && cfg.Gateway.SessionIdleTimeoutMinutes > 0 {
		defaultIdleTimeoutMinutes = cfg.Gateway.SessionIdleTimeoutMinutes
	}
	return NewSessionLimitCache(rdb, defaultIdleTimeoutMinutes)
}

// ProviderSet is the Wire provider set for all repositories
var ProviderSet = wire.NewSet(
	NewUserRepository,
	NewAPIKeyRepository,
	NewGroupRepository,
	NewAccountRepository,
	NewProxyRepository,
	NewRedeemCodeRepository,
	NewPromoCodeRepository,
	NewAnnouncementRepository,
	NewAnnouncementReadRepository,
	NewUsageLogRepository,
	NewUsageCleanupRepository,
	NewDashboardAggregationRepository,
	NewSettingRepository,
	NewOpsRepository,
	NewUserSubscriptionRepository,
	NewUserAttributeDefinitionRepository,
	NewUserAttributeValueRepository,
	NewUserGroupRateRepository,
	NewErrorPassthroughRepository,
	NewOAuthProviderRepository,
	NewUserOAuthBindingRepository,
	NewProductRepository,
	NewPaymentOrderRepository,

	// Cache implementations
	NewGatewayCache,
	NewBillingCache,
	NewAPIKeyCache,
	NewTempUnschedCache,
	NewTimeoutCounterCache,
	ProvideConcurrencyCache,
	ProvideSessionLimitCache,
	NewDashboardCache,
	NewEmailCache,
	NewIdentityCache,
	NewRedeemCache,
	NewUpdateCache,
	NewGeminiTokenCache,
	NewSchedulerCache,
	NewSchedulerOutboxRepository,
	NewProxyLatencyCache,
	NewTotpCache,
	NewRefreshTokenCache,
	NewErrorPassthroughCache,

	// Encryptors
	NewAESEncryptor,

	// HTTP service ports (DI Strategy A: return interface directly)
	NewTurnstileVerifier,
	ProvidePricingRemoteClient,
	ProvideGitHubReleaseClient,
	NewProxyExitInfoProber,
	NewClaudeUsageFetcher,
	NewClaudeOAuthClient,
	NewHTTPUpstream,
	NewOpenAIOAuthClient,
	NewGeminiOAuthClient,
	NewGeminiCliCodeAssistClient,

	ProvideEntAndSQLDB,
	ProvideEntClient,
	ProvideSQLDB,
	ProvideRedis,
)

// EntClientAndDB 包装 Ent 客户端和 SQL DB
type EntClientAndDB struct {
	Client *ent.Client
	DB     *sql.DB
}

// ProvideEntAndSQLDB 为依赖注入提供 Ent 客户端和 SQL DB。
//
// 该函数是 InitEnt 的包装器，符合 Wire 的依赖提供函数签名要求。
// Wire 会在编译时分析依赖关系，自动生成初始化代码。
//
// 依赖：config.Config
// 提供：*ent.Client, *sql.DB
func ProvideEntAndSQLDB(cfg *config.Config) (*EntClientAndDB, error) {
	client, db, err := InitEnt(cfg)
	if err != nil {
		return nil, err
	}
	return &EntClientAndDB{Client: client, DB: db}, nil
}

// ProvideEntClient 从 EntClientAndDB 提取 Ent 客户端
func ProvideEntClient(e *EntClientAndDB) *ent.Client {
	return e.Client
}

// ProvideSQLDB 从 EntClientAndDB 提取 SQL DB
func ProvideSQLDB(e *EntClientAndDB) *sql.DB {
	return e.DB
}

// ProvideRedis 为依赖注入提供 Redis 客户端。
//
// Redis 用于：
//   - 分布式锁（如并发控制）
//   - 缓存（如用户会话、API 响应缓存）
//   - 速率限制
//   - 实时统计数据
//
// 依赖：config.Config
// 提供：*redis.Client
func ProvideRedis(cfg *config.Config) *redis.Client {
	return InitRedis(cfg)
}
