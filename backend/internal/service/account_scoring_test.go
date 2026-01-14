package service

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestCalculateCapacityScore(t *testing.T) {
	tests := []struct {
		name        string
		utilization int
		hardBlocked bool
		expected    float64
		tolerance   float64 // allow floating point tolerance
	}{
		{"hard blocked", 50, true, 0, 0},
		{"negative utilization", -10, false, 100, 0},
		{"0% utilization", 0, false, 100, 0},
		{"49% utilization", 49, false, 100, 0},
		{"50% utilization", 50, false, 100, 0},
		{"65% utilization", 65, false, 75, 0},
		{"80% utilization", 80, false, 50, 0},
		{"87.5% utilization", 87, false, 31.333, 0.01},
		{"95% utilization", 95, false, 10, 0},
		{"100% utilization", 100, false, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateCapacityScore(tt.utilization, tt.hardBlocked)
			if tt.tolerance > 0 {
				if math.Abs(result-tt.expected) > tt.tolerance {
					t.Errorf("calculateCapacityScore(%d, %v) = %v, want %v (±%v)",
						tt.utilization, tt.hardBlocked, result, tt.expected, tt.tolerance)
				}
			} else if result != tt.expected {
				t.Errorf("calculateCapacityScore(%d, %v) = %v, want %v",
					tt.utilization, tt.hardBlocked, result, tt.expected)
			}
		})
	}
}

func TestCalculateLoadScore(t *testing.T) {
	tests := []struct {
		name        string
		current     int
		max         int
		expectedMin float64
		expectedMax float64
	}{
		{"no limit", 5, 0, 49, 51}, // maxConcurrency <= 0 returns 50
		{"negative current", -1, 10, 99, 101},
		{"0 load", 0, 10, 99, 101},
		{"50% load", 5, 10, 24, 26}, // (1-0.5)² = 0.25 → 25
		{"100% load", 10, 10, -1, 1},
		{"over capacity", 15, 10, -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateLoadScore(tt.current, tt.max)
			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("calculateLoadScore(%d, %d) = %v, want in range [%v, %v]",
					tt.current, tt.max, result, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestCalculateHistoryScore(t *testing.T) {
	tests := []struct {
		recent429Count int
		expected       float64
	}{
		{0, 100},
		{1, 70},
		{2, 70},
		{3, 40},
		{5, 40},
		{6, 10},
		{10, 10},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := calculateHistoryScore(tt.recent429Count)
			if result != tt.expected {
				t.Errorf("calculateHistoryScore(%d) = %v, want %v",
					tt.recent429Count, result, tt.expected)
			}
		})
	}
}

func TestCalculatePriorityScore(t *testing.T) {
	tests := []struct {
		priority int
		expected float64
	}{
		{0, 100}, // priority <= 0 treated as 1
		{1, 100},
		{2, 90},
		{5, 60},
		{10, 10},
		{11, 0},
		{15, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := calculatePriorityScore(tt.priority)
			if result != tt.expected {
				t.Errorf("calculatePriorityScore(%d) = %v, want %v",
					tt.priority, result, tt.expected)
			}
		})
	}
}

func TestSelectByWeightedRandom(t *testing.T) {
	t.Run("empty accounts", func(t *testing.T) {
		result := SelectByWeightedRandom(nil, 10)
		if result != nil {
			t.Error("Expected nil for empty accounts")
		}
	})

	t.Run("single account above threshold", func(t *testing.T) {
		accounts := []ScoredAccount{
			{Account: &Account{ID: 1}, Score: 50},
		}
		result := SelectByWeightedRandom(accounts, 10)
		if result == nil || result.Account.ID != 1 {
			t.Error("Expected single account to be returned")
		}
	})

	t.Run("single account below threshold", func(t *testing.T) {
		accounts := []ScoredAccount{
			{Account: &Account{ID: 1}, Score: 5},
		}
		result := SelectByWeightedRandom(accounts, 10)
		if result != nil {
			t.Error("Expected nil for account below threshold")
		}
	})

	t.Run("all accounts below threshold", func(t *testing.T) {
		accounts := []ScoredAccount{
			{Account: &Account{ID: 1}, Score: 5},
			{Account: &Account{ID: 2}, Score: 8},
		}
		result := SelectByWeightedRandom(accounts, 10)
		if result != nil {
			t.Error("Expected nil when all accounts below threshold")
		}
	})

	t.Run("weighted distribution", func(t *testing.T) {
		accounts := []ScoredAccount{
			{Account: &Account{ID: 1}, Score: 90},
			{Account: &Account{ID: 2}, Score: 10},
		}

		// Run multiple times to verify weighted distribution
		counts := make(map[int64]int)
		iterations := 1000
		for i := 0; i < iterations; i++ {
			result := SelectByWeightedRandom(accounts, 0)
			if result != nil {
				counts[result.Account.ID]++
			}
		}

		// Account 1 should be selected ~90% of the time
		ratio := float64(counts[1]) / float64(iterations)
		if ratio < 0.80 || ratio > 0.98 {
			t.Errorf("Expected account 1 to be selected ~90%% of time, got %.2f%%", ratio*100)
		}
	})
}

func TestSelectByHighestScore(t *testing.T) {
	t.Run("empty accounts", func(t *testing.T) {
		result := SelectByHighestScore(nil, 10)
		if result != nil {
			t.Error("Expected nil for empty accounts")
		}
	})

	t.Run("all below threshold", func(t *testing.T) {
		accounts := []ScoredAccount{
			{Account: &Account{ID: 1}, Score: 5},
			{Account: &Account{ID: 2}, Score: 8},
		}
		result := SelectByHighestScore(accounts, 10)
		if result != nil {
			t.Error("Expected nil when all below threshold")
		}
	})

	t.Run("selects highest", func(t *testing.T) {
		accounts := []ScoredAccount{
			{Account: &Account{ID: 1}, Score: 50},
			{Account: &Account{ID: 2}, Score: 80},
			{Account: &Account{ID: 3}, Score: 60},
		}
		result := SelectByHighestScore(accounts, 10)
		if result == nil || result.Account.ID != 2 {
			t.Error("Expected account 2 (highest score) to be selected")
		}
	})
}

func TestCalculateAccountScore(t *testing.T) {
	ctx := context.Background()

	account := &Account{
		ID:          1,
		Platform:    "anthropic",
		Concurrency: 10,
		Priority:    1,
	}

	scoringCtx := &AccountScoringContext{
		Now:            time.Now(),
		RequestedModel: "claude-3-opus",
		Config:         DefaultAccountScoringConfig(),
	}

	// Without quota info (non-Antigravity)
	scored := CalculateAccountScore(ctx, account, scoringCtx, nil)

	if scored.Account != account {
		t.Error("Expected scored account to match input")
	}
	if scored.CapacityScore != 100 {
		t.Errorf("Expected 100 capacity score for non-Antigravity, got %v", scored.CapacityScore)
	}
	if scored.PriorityScore != 100 {
		t.Errorf("Expected 100 priority score for priority 1, got %v", scored.PriorityScore)
	}
	if scored.HistoryScore != 100 {
		t.Errorf("Expected 100 history score with no 429s, got %v", scored.HistoryScore)
	}
}

func TestScoreAndSelectAccount(t *testing.T) {
	ctx := context.Background()

	accounts := []Account{
		{ID: 1, Platform: "anthropic", Concurrency: 10, Priority: 1},
		{ID: 2, Platform: "anthropic", Concurrency: 10, Priority: 5},
		{ID: 3, Platform: "anthropic", Concurrency: 10, Priority: 10},
	}

	scoringCtx := &AccountScoringContext{
		Now:            time.Now(),
		RequestedModel: "claude-3-opus",
		Config:         DefaultAccountScoringConfig(),
	}

	t.Run("weighted random selection", func(t *testing.T) {
		selected, allScored := ScoreAndSelectAccount(ctx, accounts, scoringCtx, nil, true)

		if selected == nil {
			t.Error("Expected an account to be selected")
		}
		if len(allScored) != 3 {
			t.Errorf("Expected 3 scored accounts, got %d", len(allScored))
		}
	})

	t.Run("highest score selection", func(t *testing.T) {
		selected, _ := ScoreAndSelectAccount(ctx, accounts, scoringCtx, nil, false)

		if selected == nil {
			t.Fatal("Expected an account to be selected")
		}
		// Account 1 has priority 1 (highest priority score)
		if selected.Account.ID != 1 {
			t.Errorf("Expected account 1 to be selected (highest priority), got %d", selected.Account.ID)
		}
	})

	t.Run("empty accounts", func(t *testing.T) {
		selected, allScored := ScoreAndSelectAccount(ctx, nil, scoringCtx, nil, true)

		if selected != nil {
			t.Error("Expected nil for empty accounts")
		}
		if allScored != nil {
			t.Error("Expected nil scored accounts for empty input")
		}
	})
}

func TestDefaultAccountScoringConfig(t *testing.T) {
	cfg := DefaultAccountScoringConfig()

	if cfg.WeightCapacity != DefaultWeightCapacity {
		t.Errorf("Expected default weight capacity %v, got %v", DefaultWeightCapacity, cfg.WeightCapacity)
	}
	if cfg.WeightLoad != DefaultWeightLoad {
		t.Errorf("Expected default weight load %v, got %v", DefaultWeightLoad, cfg.WeightLoad)
	}
	if cfg.WeightHistory != DefaultWeightHistory {
		t.Errorf("Expected default weight history %v, got %v", DefaultWeightHistory, cfg.WeightHistory)
	}
	if cfg.WeightPriority != DefaultWeightPriority {
		t.Errorf("Expected default weight priority %v, got %v", DefaultWeightPriority, cfg.WeightPriority)
	}

	// Verify weights sum to 1.0 (with floating point tolerance)
	total := cfg.WeightCapacity + cfg.WeightLoad + cfg.WeightHistory + cfg.WeightPriority
	if math.Abs(total-1.0) > 1e-9 {
		t.Errorf("Expected weights to sum to 1.0, got %v", total)
	}
}
