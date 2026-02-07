package service

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Backoff constants for quota-related 429s
const (
	quotaBackoffBase = 1 * time.Second  // Initial backoff: 1s
	quotaBackoffMax  = 30 * time.Minute // Max backoff: 30min
	cleanupInterval  = 5 * time.Minute  // Cleanup interval for expired entries
)

// Model429State tracks 429 state for a specific (accountID, model) pair
type Model429State struct {
	Unavailable    bool      // Whether this account-model is currently unavailable
	NextRetryAfter time.Time // When to retry
	BackoffLevel   int       // Current backoff level for exponential backoff
	RecordedAt     time.Time // When this record was created
}

// Model429Tracker provides model-level 429 tracking with exponential backoff
// Key: accountID -> model -> state
type Model429Tracker struct {
	mu     sync.RWMutex
	states map[int64]map[string]*Model429State
}

// NewModel429Tracker creates a new model-level 429 tracker
func NewModel429Tracker() *Model429Tracker {
	t := &Model429Tracker{
		states: make(map[int64]map[string]*Model429State),
	}
	// 启动后台清理 goroutine 防止内存泄漏
	go t.startCleanupLoop()
	return t
}

// startCleanupLoop periodically clears expired entries to prevent memory leaks
func (t *Model429Tracker) startCleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		cleared := t.ClearExpired()
		if cleared > 0 {
			log.Printf("[Model429Tracker] Cleared %d expired entries", cleared)
		}
	}
}

// nextQuotaCooldown calculates the next cooldown duration using exponential backoff
// Returns (cooldown duration, new backoff level)
func nextQuotaCooldown(prevLevel int) (time.Duration, int) {
	// cooldown = base * 2^level (1s, 2s, 4s, 8s, ... up to 30min)
	cooldown := quotaBackoffBase * time.Duration(1<<prevLevel)
	if cooldown >= quotaBackoffMax {
		return quotaBackoffMax, prevLevel // Cap at max, don't increase level
	}
	return cooldown, prevLevel + 1
}

// Record429 records a 429 error for the given account-model pair
// upstreamResetAt is the reset time from upstream (if available), otherwise zero
func (t *Model429Tracker) Record429(accountID int64, model string, upstreamResetAt time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get or create account map
	accountStates, exists := t.states[accountID]
	if !exists {
		accountStates = make(map[string]*Model429State)
		t.states[accountID] = accountStates
	}

	// Get or create model state
	state, exists := accountStates[model]
	if !exists {
		state = &Model429State{
			BackoffLevel: 0,
		}
		accountStates[model] = state
	}

	now := time.Now()

	// If upstream provided a valid reset time, use it directly (no backoff)
	// This handles temporary rate limits (RATE_LIMIT_EXCEEDED) which have short reset times
	if !upstreamResetAt.IsZero() && upstreamResetAt.After(now) {
		state.Unavailable = true
		state.NextRetryAfter = upstreamResetAt
		// Don't increase backoff level for upstream-specified resets
		// as they are authoritative and not our fallback
		state.RecordedAt = now
		return
	}

	// No valid upstream reset time - use exponential backoff
	// This handles cases where we can't parse the upstream response
	cooldown, newLevel := nextQuotaCooldown(state.BackoffLevel)
	state.Unavailable = true
	state.NextRetryAfter = now.Add(cooldown)
	state.BackoffLevel = newLevel
	state.RecordedAt = now
}

// IsAccountAvailableForModel checks if an account is available for a specific model
func (t *Model429Tracker) IsAccountAvailableForModel(accountID int64, model string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	accountStates, exists := t.states[accountID]
	if !exists {
		return true
	}

	state, exists := accountStates[model]
	if !exists {
		return true
	}

	if !state.Unavailable {
		return true
	}

	// Check if cooldown has expired
	return time.Now().After(state.NextRetryAfter)
}

// MarkAvailable marks an account-model pair as available (successful request)
// This resets the backoff level
func (t *Model429Tracker) MarkAvailable(accountID int64, model string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	accountStates, exists := t.states[accountID]
	if !exists {
		return
	}

	state, exists := accountStates[model]
	if !exists {
		return
	}

	state.Unavailable = false
	state.BackoffLevel = 0
}

// GetNextRetryTime returns the next retry time for an account-model pair
// Returns nil if not in 429 state or already available
func (t *Model429Tracker) GetNextRetryTime(accountID int64, model string) *time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()

	accountStates, exists := t.states[accountID]
	if !exists {
		return nil
	}

	state, exists := accountStates[model]
	if !exists {
		return nil
	}

	if !state.Unavailable {
		return nil
	}

	if time.Now().After(state.NextRetryAfter) {
		return nil // Already available
	}

	return &state.NextRetryAfter
}

// GetEarliestRetryTime returns the earliest retry time among the given accounts for a model
// Returns nil if any account is immediately available
func (t *Model429Tracker) GetEarliestRetryTime(accountIDs []int64, model string) *time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var earliest *time.Time
	now := time.Now()

	for _, accountID := range accountIDs {
		accountStates, exists := t.states[accountID]
		if !exists {
			return nil // Account not tracked = immediately available
		}

		state, exists := accountStates[model]
		if !exists {
			return nil // Model not tracked for this account = immediately available
		}

		if !state.Unavailable {
			return nil // Account is available
		}

		if now.After(state.NextRetryAfter) {
			return nil // Cooldown expired = immediately available
		}

		if earliest == nil || state.NextRetryAfter.Before(*earliest) {
			earliest = &state.NextRetryAfter
		}
	}

	return earliest
}

// ClearExpired removes expired records to prevent memory leaks
// Returns the number of cleared entries
func (t *Model429Tracker) ClearExpired() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	cleared := 0

	for accountID, accountStates := range t.states {
		for model, state := range accountStates {
			if now.After(state.NextRetryAfter) {
				delete(accountStates, model)
				cleared++
			}
		}
		// Remove empty account maps to prevent memory leak
		if len(accountStates) == 0 {
			delete(t.states, accountID)
		}
	}

	return cleared
}

// GetStats returns current tracker statistics for monitoring
func (t *Model429Tracker) GetStats() map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	totalAccounts := len(t.states)
	totalEntries := 0
	unavailableCount := 0
	now := time.Now()

	for _, accountStates := range t.states {
		for _, state := range accountStates {
			totalEntries++
			if state.Unavailable && now.Before(state.NextRetryAfter) {
				unavailableCount++
			}
		}
	}

	return map[string]any{
		"total_accounts":    totalAccounts,
		"total_entries":     totalEntries,
		"unavailable_count": unavailableCount,
	}
}

// String returns a debug string representation
func (t *Model429Tracker) String() string {
	stats := t.GetStats()
	return fmt.Sprintf("Model429Tracker{accounts=%d, entries=%d, unavailable=%d}",
		stats["total_accounts"], stats["total_entries"], stats["unavailable_count"])
}
