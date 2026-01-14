package service

import (
	"context"
	"math/rand"
	"time"
)

// 默认评分权重
const (
	DefaultWeightCapacity  = 0.30
	DefaultWeightLoad      = 0.25
	DefaultWeightHistory   = 0.20
	DefaultWeightPriority  = 0.10
	DefaultWeightFreshness = 0.10 // 最久未用优先
	DefaultWeightOAuth     = 0.05 // Gemini 平台 OAuth 偏好
)

// AccountScoringConfig 评分配置
type AccountScoringConfig struct {
	WeightCapacity    float64 // 配额容量权重，默认 0.30
	WeightLoad        float64 // 实时负载权重，默认 0.25
	WeightHistory     float64 // 历史表现权重，默认 0.20
	WeightPriority    float64 // 优先级权重，默认 0.10
	WeightFreshness   float64 // 新鲜度权重（最久未用优先），默认 0.10
	WeightOAuth       float64 // OAuth 偏好权重（仅 Gemini），默认 0.05
	MinScoreThreshold float64 // 最低分数阈值，低于此分数不参与选择，默认 10
}

// DefaultAccountScoringConfig 返回默认配置
func DefaultAccountScoringConfig() *AccountScoringConfig {
	return &AccountScoringConfig{
		WeightCapacity:    DefaultWeightCapacity,
		WeightLoad:        DefaultWeightLoad,
		WeightHistory:     DefaultWeightHistory,
		WeightPriority:    DefaultWeightPriority,
		WeightFreshness:   DefaultWeightFreshness,
		WeightOAuth:       DefaultWeightOAuth,
		MinScoreThreshold: 10,
	}
}

// ScoredAccount 带评分的账户
type ScoredAccount struct {
	Account *Account
	Score   float64

	// 评分明细（用于调试/日志）
	CapacityScore  float64
	LoadScore      float64
	HistoryScore   float64
	PriorityScore  float64
	FreshnessScore float64 // 新鲜度分数（最久未用优先）
	OAuthScore     float64 // OAuth 偏好分数（仅 Gemini 平台生效）
}

// AccountScoringContext 评分计算上下文
type AccountScoringContext struct {
	Now              time.Time
	RequestedModel   string
	Tracker          *Account429Tracker
	Config           *AccountScoringConfig
	ConcurrencyCache ConcurrencyCache // 用于获取实时负载
	Platform         string           // 目标平台（用于 OAuth 偏好判断）
	PreferOAuth      bool             // 是否偏好 OAuth 账户（Gemini 平台）
}

// calculateCapacityScore 计算配额容量分 [0-100]
// 基于配额使用率：<50% → 100, 50-80% → 线性到50, 80-95% → 线性到10, >95% → 10
func calculateCapacityScore(utilization int, hardBlocked bool) float64 {
	if hardBlocked {
		return 0
	}
	if utilization < 0 {
		utilization = 0
	}
	if utilization < 50 {
		return 100
	}
	if utilization < 80 {
		// 50-80%: 线性从 100 降到 50
		return 100 - float64(utilization-50)*50/30
	}
	if utilization < 95 {
		// 80-95%: 线性从 50 降到 10
		return 50 - float64(utilization-80)*40/15
	}
	// >= 95%
	return 10
}

// calculateCapacityScoreWithReset 计算配额容量分，包含 reset_time 调整 [0-100]
// 当 utilization >= 阈值（快满）时，reset_time 越早，分数越高
// resetAt: 配额重置时间；now: 当前时间；hasReset: 是否有有效的重置时间
func calculateCapacityScoreWithReset(utilization int, hardBlocked bool, resetAt time.Time, hasReset bool, now time.Time) float64 {
	baseScore := calculateCapacityScore(utilization, hardBlocked)

	// 只有快满（utilization >= 90）且有有效 reset_time 时才进行调整
	if utilization < 90 || !hasReset {
		return baseScore
	}

	// 计算 reset_time 距离现在的时间（分钟）
	minutesUntilReset := resetAt.Sub(now).Minutes()

	// 根据 reset_time 调整分数：0-30分钟内 → +5分，30-120分钟 → 线性到0分
	// 这样可以区分快满但即将重置的账户
	if minutesUntilReset <= 0 {
		// 已过重置时间，给最大加分
		return baseScore + 5
	}
	if minutesUntilReset <= 30 {
		// 30分钟内重置，给满分加分
		return baseScore + 5
	}
	if minutesUntilReset <= 120 {
		// 30-120分钟：线性从 5 降到 0
		bonus := 5 * (1 - (minutesUntilReset-30)/90)
		return baseScore + bonus
	}
	// > 120 分钟，无加分
	return baseScore
}

// calculateLoadScore 计算实时负载分 [0-100]
// 基于当前并发数/最大并发数，使用二次衰减优先低负载
func calculateLoadScore(currentConnections, maxConcurrency int) float64 {
	if maxConcurrency <= 0 {
		// 无限制时返回中间值
		return 50
	}
	if currentConnections < 0 {
		currentConnections = 0
	}

	loadRatio := float64(currentConnections) / float64(maxConcurrency)
	if loadRatio >= 1 {
		return 0
	}

	// 二次衰减：(1-loadRatio)²
	remaining := 1 - loadRatio
	return 100 * remaining * remaining
}

// calculateHistoryScore 计算历史表现分 [0-100]
// 基于最近历史窗口内的 429 次数
func calculateHistoryScore(recent429Count int) float64 {
	switch {
	case recent429Count == 0:
		return 100
	case recent429Count <= 2:
		return 70
	case recent429Count <= 5:
		return 40
	default:
		return 10
	}
}

// calculatePriorityScore 计算优先级分 [0-100]
// 基于账户配置的优先级（1最高，10+最低）
func calculatePriorityScore(priority int) float64 {
	if priority <= 0 {
		priority = 1
	}
	score := 100 - (priority-1)*10
	if score < 0 {
		score = 0
	}
	return float64(score)
}

// calculateFreshnessScore 计算新鲜度分 [0-100]
// 最久未用的账户得分更高，避免所有请求集中到同一账户
func calculateFreshnessScore(lastUsedAt *time.Time, now time.Time) float64 {
	if lastUsedAt == nil {
		// 从未使用过的账户优先级最高
		return 100
	}

	// 计算距离上次使用的时间（分钟）
	elapsedMinutes := now.Sub(*lastUsedAt).Minutes()

	// 线性映射：0分钟 → 0分，120分钟（2小时）及以上 → 100分
	// 这样可以精确区分 -1h 和 -2h 的账户
	if elapsedMinutes >= 120 {
		return 100
	}
	if elapsedMinutes <= 0 {
		return 0
	}
	return elapsedMinutes * 100 / 120
}

// calculateOAuthPreferenceScore 计算 OAuth 偏好分 [0-100]
// 仅在 Gemini 平台生效，OAuth 账户优先于 API Key
func calculateOAuthPreferenceScore(accountType string, preferOAuth bool) float64 {
	if !preferOAuth {
		// 非 Gemini 平台或不需要 OAuth 偏好，返回中立值
		return 50
	}
	if accountType == AccountTypeOAuth {
		return 100
	}
	return 0
}

// CalculateAccountScore 计算账户综合评分
func CalculateAccountScore(
	ctx context.Context,
	account *Account,
	scoringCtx *AccountScoringContext,
	quotaInfo *AntigravityQuotaKey, // 配额信息（可选，仅 Antigravity 平台）
) *ScoredAccount {
	cfg := scoringCtx.Config
	if cfg == nil {
		cfg = DefaultAccountScoringConfig()
	}

	scored := &ScoredAccount{
		Account: account,
	}

	// 1. 配额容量分
	if quotaInfo != nil {
		scored.CapacityScore = calculateCapacityScoreWithReset(
			quotaInfo.utilization, quotaInfo.hardBlocked,
			quotaInfo.resetAt, quotaInfo.hasReset, scoringCtx.Now,
		)
	} else {
		// 非 Antigravity 平台或无配额信息，给予满分
		scored.CapacityScore = 100
	}

	// 2. 实时负载分
	currentLoad := 0
	if scoringCtx.ConcurrencyCache != nil && account.Concurrency > 0 {
		if load, err := scoringCtx.ConcurrencyCache.GetAccountConcurrency(ctx, account.ID); err == nil {
			currentLoad = load
		}
	}
	scored.LoadScore = calculateLoadScore(currentLoad, account.Concurrency)

	// 3. 历史表现分
	recent429Count := 0
	if scoringCtx.Tracker != nil {
		// 使用映射后的模型名
		mappedModel := scoringCtx.RequestedModel
		if account.Platform == PlatformAntigravity {
			mappedModel = (&AntigravityGatewayService{}).getMappedModel(account, scoringCtx.RequestedModel)
		}
		recent429Count = scoringCtx.Tracker.GetRecent429CountAt(account.ID, mappedModel, scoringCtx.Now)
	}
	scored.HistoryScore = calculateHistoryScore(recent429Count)

	// 4. 优先级分
	scored.PriorityScore = calculatePriorityScore(account.Priority)

	// 5. 新鲜度分（最久未用优先）
	scored.FreshnessScore = calculateFreshnessScore(account.LastUsedAt, scoringCtx.Now)

	// 6. OAuth 偏好分（仅 Gemini 平台）
	scored.OAuthScore = calculateOAuthPreferenceScore(account.Type, scoringCtx.PreferOAuth)

	// 7. 综合评分
	scored.Score = cfg.WeightCapacity*scored.CapacityScore +
		cfg.WeightLoad*scored.LoadScore +
		cfg.WeightHistory*scored.HistoryScore +
		cfg.WeightPriority*scored.PriorityScore +
		cfg.WeightFreshness*scored.FreshnessScore +
		cfg.WeightOAuth*scored.OAuthScore

	return scored
}

// SelectByWeightedRandom 加权随机选择账户
// 按分数加权概率选择，高分账户被选中概率更高，但不是100%
// 自然分散并发请求，避免惊群效应
func SelectByWeightedRandom(accounts []ScoredAccount, minScoreThreshold float64) *ScoredAccount {
	if len(accounts) == 0 {
		return nil
	}
	if len(accounts) == 1 {
		// 单账户时直接返回（不检查阈值，避免完全无可用账户）
		return &accounts[0]
	}

	// 过滤低于阈值的账户
	var eligible []ScoredAccount
	for _, acc := range accounts {
		if acc.Score >= minScoreThreshold {
			eligible = append(eligible, acc)
		}
	}

	// 安全机制：如果所有账户都低于阈值，降级到选择最高分账户
	// 避免 "no available accounts" 错误
	if len(eligible) == 0 {
		var best *ScoredAccount
		for i := range accounts {
			if best == nil || accounts[i].Score > best.Score {
				best = &accounts[i]
			}
		}
		return best
	}
	if len(eligible) == 1 {
		return &eligible[0]
	}

	// 计算总分
	totalScore := 0.0
	for _, acc := range eligible {
		totalScore += acc.Score
	}

	if totalScore <= 0 {
		// 所有账户分数为 0 或负数，随机选择
		idx := rand.Intn(len(eligible))
		return &eligible[idx]
	}

	// 加权随机选择
	r := rand.Float64() * totalScore
	cumulative := 0.0
	for i := range eligible {
		cumulative += eligible[i].Score
		if r <= cumulative {
			return &eligible[i]
		}
	}

	// 兜底返回最后一个
	return &eligible[len(eligible)-1]
}

// SelectByHighestScore 选择最高分账户（确定性选择，用于禁用加权随机时）
func SelectByHighestScore(accounts []ScoredAccount, minScoreThreshold float64) *ScoredAccount {
	if len(accounts) == 0 {
		return nil
	}

	var best *ScoredAccount
	var bestUnfiltered *ScoredAccount // 用于安全机制回退

	for i := range accounts {
		// 跟踪最高分（忽略阈值）
		if bestUnfiltered == nil || accounts[i].Score > bestUnfiltered.Score {
			bestUnfiltered = &accounts[i]
		}
		// 跟踪高于阈值的最高分
		if accounts[i].Score >= minScoreThreshold {
			if best == nil || accounts[i].Score > best.Score {
				best = &accounts[i]
			}
		}
	}

	// 安全机制：如果没有高于阈值的账户，返回最高分账户
	// 避免 "no available accounts" 错误
	if best == nil {
		return bestUnfiltered
	}

	return best
}

// ScoreAndSelectAccount 评分并选择账户（完整流程）
// 返回选中的账户和所有评分结果（用于调试/日志）
func ScoreAndSelectAccount(
	ctx context.Context,
	accounts []Account,
	scoringCtx *AccountScoringContext,
	quotaProvider func(*Account) *AntigravityQuotaKey,
	useWeightedRandom bool,
) (*ScoredAccount, []ScoredAccount) {
	if len(accounts) == 0 {
		return nil, nil
	}

	cfg := scoringCtx.Config
	if cfg == nil {
		cfg = DefaultAccountScoringConfig()
	}

	// 计算所有账户评分
	scoredAccounts := make([]ScoredAccount, 0, len(accounts))
	for i := range accounts {
		var quotaInfo *AntigravityQuotaKey
		if quotaProvider != nil {
			quotaInfo = quotaProvider(&accounts[i])
		}
		scored := CalculateAccountScore(ctx, &accounts[i], scoringCtx, quotaInfo)
		scoredAccounts = append(scoredAccounts, *scored)
	}

	// 选择账户
	var selected *ScoredAccount
	if useWeightedRandom {
		selected = SelectByWeightedRandom(scoredAccounts, cfg.MinScoreThreshold)
	} else {
		selected = SelectByHighestScore(scoredAccounts, cfg.MinScoreThreshold)
	}

	return selected, scoredAccounts
}
