package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// TokenRefreshService OAuth token自动刷新服务
// 定期检查并刷新即将过期的token
type TokenRefreshService struct {
	accountRepo AccountRepository
	refreshers  []TokenRefresher
	cfg         *config.TokenRefreshConfig
	rootCfg     *config.Config

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewTokenRefreshService 创建token刷新服务
func NewTokenRefreshService(
	accountRepo AccountRepository,
	oauthService *OAuthService,
	openaiOAuthService *OpenAIOAuthService,
	geminiOAuthService *GeminiOAuthService,
	antigravityOAuthService *AntigravityOAuthService,
	cfg *config.Config,
) *TokenRefreshService {
	s := &TokenRefreshService{
		accountRepo: accountRepo,
		cfg:         &cfg.TokenRefresh,
		rootCfg:     cfg,
		stopCh:      make(chan struct{}),
	}

	// 注册平台特定的刷新器
	s.refreshers = []TokenRefresher{
		NewClaudeTokenRefresher(oauthService),
		NewOpenAITokenRefresher(openaiOAuthService),
		NewGeminiTokenRefresher(geminiOAuthService),
		NewAntigravityTokenRefresher(antigravityOAuthService),
	}

	return s
}

// Start 启动后台刷新服务
func (s *TokenRefreshService) Start() {
	if s == nil {
		return
	}

	startedAny := false

	// 1) Token refresh loop（与历史行为保持一致）
	if s.cfg != nil && s.cfg.Enabled {
		s.wg.Add(1)
		go s.refreshLoop()
		startedAny = true

		log.Printf("[TokenRefresh] Service started (check every %d minutes, refresh %v hours before expiry)",
			s.cfg.CheckIntervalMinutes, s.cfg.RefreshBeforeExpiryHours)
	} else {
		log.Println("[TokenRefresh] Service disabled by configuration")
	}

	// 2) Antigravity quota snapshot loop（方案D：配额快照 → 调度避让）
	agQuotaCfg := s.antigravityQuotaSchedulingConfig()
	if agQuotaCfg.Enabled {
		s.wg.Add(1)
		go s.antigravityQuotaSnapshotLoop()
		startedAny = true

		log.Printf("[AntigravityQuotaSnapshot] Service started (interval=%ds concurrency=%d soft_threshold=%d stale_after=%ds)",
			agQuotaCfg.RefreshIntervalSeconds,
			agQuotaCfg.FetchConcurrency,
			agQuotaCfg.SoftUtilizationThreshold,
			agQuotaCfg.StaleAfterSeconds,
		)
	} else {
		log.Println("[AntigravityQuotaSnapshot] Service disabled by configuration")
	}

	if !startedAny {
		log.Println("[Background] No background services started (token_refresh disabled and antigravity quota snapshot disabled)")
	}
}

// Stop 停止刷新服务
func (s *TokenRefreshService) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	log.Println("[TokenRefresh] Service stopped")
}

func (s *TokenRefreshService) antigravityQuotaSchedulingConfig() config.AntigravityQuotaSchedulingConfig {
	if s == nil || s.rootCfg == nil {
		return config.AntigravityQuotaSchedulingConfig{Enabled: false}
	}
	return s.rootCfg.Gateway.Scheduling.AntigravityQuota
}

// refreshLoop 刷新循环
func (s *TokenRefreshService) refreshLoop() {
	defer s.wg.Done()

	// 计算检查间隔
	checkInterval := time.Duration(s.cfg.CheckIntervalMinutes) * time.Minute
	if checkInterval < time.Minute {
		checkInterval = 5 * time.Minute
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// 启动时立即执行一次检查
	s.processRefresh()

	for {
		select {
		case <-ticker.C:
			s.processRefresh()
		case <-s.stopCh:
			return
		}
	}
}

func (s *TokenRefreshService) antigravityQuotaSnapshotLoop() {
	defer s.wg.Done()

	cfg := s.antigravityQuotaSchedulingConfig()
	if !cfg.Enabled {
		return
	}

	interval := time.Duration(cfg.RefreshIntervalSeconds) * time.Second
	if interval < time.Minute {
		// 安全兜底：避免配置过小导致过度抓取
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 启动时立即抓取一次，尽快让调度拥有“已满/快满”的视野
	s.processAntigravityQuotaSnapshots()

	for {
		select {
		case <-ticker.C:
			s.processAntigravityQuotaSnapshots()
		case <-s.stopCh:
			return
		}
	}
}

func (s *TokenRefreshService) processAntigravityQuotaSnapshots() {
	cfg := s.antigravityQuotaSchedulingConfig()
	if !cfg.Enabled || s.accountRepo == nil {
		return
	}

	start := time.Now()
	ctx := context.Background()

	// 仅抓取可调度账号：减少无意义请求
	accounts, err := s.accountRepo.ListSchedulableByPlatform(ctx, PlatformAntigravity)
	if err != nil {
		log.Printf("[AntigravityQuotaSnapshot] Failed to list accounts: %v", err)
		return
	}
	if len(accounts) == 0 {
		return
	}

	fetcher := &AntigravityQuotaFetcher{}

	concurrency := cfg.FetchConcurrency
	if concurrency <= 0 {
		concurrency = 3
	}

	type counters struct {
		total    int
		skipped  int
		fetched  int
		updated  int
		failed   int
		emptyRaw int
	}

	var mu sync.Mutex
	stats := counters{total: len(accounts)}

	jobs := make(chan *Account)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for acc := range jobs {
			// 每个账号单独超时，避免个别网络问题拖慢整轮
			reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err := s.fetchAndStoreAntigravityQuotaSnapshot(reqCtx, fetcher, acc)
			cancel()

			mu.Lock()
			if err != nil {
				stats.failed++
			} else {
				stats.updated++
			}
			mu.Unlock()
		}
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker()
	}

	for i := range accounts {
		acc := &accounts[i]
		if acc == nil {
			continue
		}
		if !fetcher.CanFetch(acc) {
			mu.Lock()
			stats.skipped++
			mu.Unlock()
			continue
		}
		mu.Lock()
		stats.fetched++
		mu.Unlock()
		jobs <- acc
	}
	close(jobs)
	wg.Wait()

	log.Printf("[AntigravityQuotaSnapshot] Cycle complete: total=%d, fetched=%d, skipped=%d, updated=%d, failed=%d, took=%s",
		stats.total, stats.fetched, stats.skipped, stats.updated, stats.failed, time.Since(start))
}

func (s *TokenRefreshService) fetchAndStoreAntigravityQuotaSnapshot(ctx context.Context, fetcher *AntigravityQuotaFetcher, account *Account) error {
	if s == nil || s.accountRepo == nil || fetcher == nil || account == nil {
		return errors.New("invalid quota snapshot dependencies")
	}
	if account.ID <= 0 {
		return errors.New("invalid account id")
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	result, err := fetcher.FetchQuota(ctx, account, proxyURL)
	if err != nil {
		// 不写回 DB，保留上一份可用快照
		return fmt.Errorf("fetch quota failed: %w", err)
	}

	// 写入 accounts.extra：结构化快照，供调度读取（避免每次调度都打上游）
	// 注意：数值会在 JSON 反序列化后变为 float64，因此读取端需要容错解析。
	now := time.Now().UTC()
	models := make(map[string]any)
	if result != nil && result.UsageInfo != nil {
		for modelName, q := range result.UsageInfo.AntigravityQuota {
			if strings.TrimSpace(modelName) == "" || q == nil {
				continue
			}
			models[modelName] = map[string]any{
				"utilization": q.Utilization,
				"reset_time":   q.ResetTime,
			}
		}
	}
	snapshot := map[string]any{
		"updated_at": now.Format(time.RFC3339),
		"models":     models,
	}
	updates := map[string]any{
		"antigravity_quota_snapshot": snapshot,
	}

	if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err != nil {
		return fmt.Errorf("update extra failed: %w", err)
	}

	return nil
}

// processRefresh 执行一次刷新检查
func (s *TokenRefreshService) processRefresh() {
	ctx := context.Background()

	// 计算刷新窗口
	refreshWindow := time.Duration(s.cfg.RefreshBeforeExpiryHours * float64(time.Hour))

	// 获取所有active状态的账号
	accounts, err := s.listActiveAccounts(ctx)
	if err != nil {
		log.Printf("[TokenRefresh] Failed to list accounts: %v", err)
		return
	}

	totalAccounts := len(accounts)
	oauthAccounts := 0 // 可刷新的OAuth账号数
	needsRefresh := 0  // 需要刷新的账号数
	refreshed, failed := 0, 0

	for i := range accounts {
		account := &accounts[i]

		// 遍历所有刷新器，找到能处理此账号的
		for _, refresher := range s.refreshers {
			if !refresher.CanRefresh(account) {
				continue
			}

			oauthAccounts++

			// 检查是否需要刷新
			if !refresher.NeedsRefresh(account, refreshWindow) {
				break // 不需要刷新，跳过
			}

			needsRefresh++

			// 执行刷新
			if err := s.refreshWithRetry(ctx, account, refresher); err != nil {
				log.Printf("[TokenRefresh] Account %d (%s) failed: %v", account.ID, account.Name, err)
				failed++
			} else {
				log.Printf("[TokenRefresh] Account %d (%s) refreshed successfully", account.ID, account.Name)
				refreshed++
			}

			// 每个账号只由一个refresher处理
			break
		}
	}

	// 始终打印周期日志，便于跟踪服务运行状态
	log.Printf("[TokenRefresh] Cycle complete: total=%d, oauth=%d, needs_refresh=%d, refreshed=%d, failed=%d",
		totalAccounts, oauthAccounts, needsRefresh, refreshed, failed)
}

// listActiveAccounts 获取所有active状态的账号
// 使用ListActive确保刷新所有活跃账号的token（包括临时禁用的）
func (s *TokenRefreshService) listActiveAccounts(ctx context.Context) ([]Account, error) {
	return s.accountRepo.ListActive(ctx)
}

// refreshWithRetry 带重试的刷新
func (s *TokenRefreshService) refreshWithRetry(ctx context.Context, account *Account, refresher TokenRefresher) error {
	var lastErr error

	for attempt := 1; attempt <= s.cfg.MaxRetries; attempt++ {
		newCredentials, err := refresher.Refresh(ctx, account)
		if err == nil {
			// 刷新成功，更新账号credentials
			account.Credentials = newCredentials
			if err := s.accountRepo.Update(ctx, account); err != nil {
				return fmt.Errorf("failed to save credentials: %w", err)
			}
			return nil
		}

		// Antigravity 账户：不可重试错误直接标记 error 状态并返回
		if account.Platform == PlatformAntigravity && isNonRetryableRefreshError(err) {
			errorMsg := fmt.Sprintf("Token refresh failed (non-retryable): %v", err)
			if setErr := s.accountRepo.SetError(ctx, account.ID, errorMsg); setErr != nil {
				log.Printf("[TokenRefresh] Failed to set error status for account %d: %v", account.ID, setErr)
			}
			return err
		}

		lastErr = err
		log.Printf("[TokenRefresh] Account %d attempt %d/%d failed: %v",
			account.ID, attempt, s.cfg.MaxRetries, err)

		// 如果还有重试机会，等待后重试
		if attempt < s.cfg.MaxRetries {
			// 指数退避：2^(attempt-1) * baseSeconds
			backoff := time.Duration(s.cfg.RetryBackoffSeconds) * time.Second * time.Duration(1<<(attempt-1))
			time.Sleep(backoff)
		}
	}

	// Antigravity 账户：其他错误仅记录日志，不标记 error（可能是临时网络问题）
	// 其他平台账户：重试失败后标记 error
	if account.Platform == PlatformAntigravity {
		log.Printf("[TokenRefresh] Account %d: refresh failed after %d retries: %v", account.ID, s.cfg.MaxRetries, lastErr)
	} else {
		errorMsg := fmt.Sprintf("Token refresh failed after %d retries: %v", s.cfg.MaxRetries, lastErr)
		if err := s.accountRepo.SetError(ctx, account.ID, errorMsg); err != nil {
			log.Printf("[TokenRefresh] Failed to set error status for account %d: %v", account.ID, err)
		}
	}

	return lastErr
}

// isNonRetryableRefreshError 判断是否为不可重试的刷新错误
// 这些错误通常表示凭证已失效，需要用户重新授权
func isNonRetryableRefreshError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	nonRetryable := []string{
		"invalid_grant",       // refresh_token 已失效
		"invalid_client",      // 客户端配置错误
		"unauthorized_client", // 客户端未授权
		"access_denied",       // 访问被拒绝
	}
	for _, needle := range nonRetryable {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}
