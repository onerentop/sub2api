package service

import (
	"context"
	"math/rand"
	"time"
)

// 默认评分权重
const (
	DefaultWeightCapacity = 0.35
	DefaultWeightLoad     = 0.30
	DefaultWeightHistory  = 0.25
	DefaultWeightPriority = 0.10
)

// AccountScoringConfig 评分配置
type AccountScoringConfig struct {
	WeightCapacity    float64 // 配额容量权重，默认 0.35
	WeightLoad        float64 // 实时负载权重，默认 0.30
	WeightHistory     float64 // 历史表现权重，默认 0.25
	WeightPriority    float64 // 优先级权重，默认 0.10
	MinScoreThreshold float64 // 最低分数阈值，低于此分数不参与选择，默认 10
}

// DefaultAccountScoringConfig 返回默认配置
func DefaultAccountScoringConfig() *AccountScoringConfig {
	return &AccountScoringConfig{
		WeightCapacity:    DefaultWeightCapacity,
		WeightLoad:        DefaultWeightLoad,
		WeightHistory:     DefaultWeightHistory,
		WeightPriority:    DefaultWeightPriority,
		MinScoreThreshold: 10,
	}
}

// ScoredAccount 带评分的账户
type ScoredAccount struct {
	Account *Account
	Score   float64

	// 评分明细（用于调试/日志）
	CapacityScore float64
	LoadScore     float64
	HistoryScore  float64
	PriorityScore float64
}

// AccountScoringContext 评分计算上下文
type AccountScoringContext struct {
	Now              time.Time
	RequestedModel   string
	Tracker          *Account429Tracker
	Config           *AccountScoringConfig
	ConcurrencyCache ConcurrencyCache // 用于获取实时负载
}

// calculateCapacityScore 计算配额容量分 [0-100]
// 基于配额使用率：<50% → 100, 50-80% → 线性到50, 80-95% → 线性到10, >95% → 0
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
		scored.CapacityScore = calculateCapacityScore(quotaInfo.utilization, quotaInfo.hardBlocked)
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

	// 5. 综合评分
	scored.Score = cfg.WeightCapacity*scored.CapacityScore +
		cfg.WeightLoad*scored.LoadScore +
		cfg.WeightHistory*scored.HistoryScore +
		cfg.WeightPriority*scored.PriorityScore

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
		if accounts[0].Score >= minScoreThreshold {
			return &accounts[0]
		}
		return nil
	}

	// 过滤低于阈值的账户
	var eligible []ScoredAccount
	for _, acc := range accounts {
		if acc.Score >= minScoreThreshold {
			eligible = append(eligible, acc)
		}
	}

	if len(eligible) == 0 {
		return nil
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
	for i := range accounts {
		if accounts[i].Score < minScoreThreshold {
			continue
		}
		if best == nil || accounts[i].Score > best.Score {
			best = &accounts[i]
		}
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
