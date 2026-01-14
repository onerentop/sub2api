package service

import (
	"sync"
	"testing"
	"time"
)

func TestAccount429Tracker_Record429_ExponentialBackoff(t *testing.T) {
	tracker := NewAccount429Tracker(&Account429TrackerConfig{
		BackoffBaseSeconds:   1, // 1s base for faster tests
		BackoffMaxSeconds:    60,
		BackoffMaxExponent:   6,
		HistoryWindowMinutes: 5,
	})

	accountID := int64(1)
	model := "claude-3-opus"

	// First 429: should have 1s backoff
	tracker.Record429(accountID, model)
	backoff1 := tracker.GetBackoffUntil(accountID, model)
	if backoff1.IsZero() {
		t.Error("Expected non-zero backoff after first 429")
	}

	// Wait a bit and record second 429: should have 2s backoff
	time.Sleep(10 * time.Millisecond)
	tracker.Record429(accountID, model)
	backoff2 := tracker.GetBackoffUntil(accountID, model)
	if !backoff2.After(backoff1) {
		t.Error("Expected longer backoff after second 429")
	}

	// Record more 429s to test exponential growth
	for i := 0; i < 5; i++ {
		time.Sleep(10 * time.Millisecond)
		tracker.Record429(accountID, model)
	}

	// After 7 429s, should be capped at max
	count := tracker.GetRecent429Count(accountID, model)
	if count != 7 {
		t.Errorf("Expected 7 429s in window, got %d", count)
	}
}

func TestAccount429Tracker_ShouldSkip(t *testing.T) {
	tracker := NewAccount429Tracker(&Account429TrackerConfig{
		BackoffBaseSeconds:   1,
		BackoffMaxSeconds:    60,
		BackoffMaxExponent:   6,
		HistoryWindowMinutes: 5,
	})

	accountID := int64(1)
	model := "claude-3-opus"

	// Before any 429: should not skip
	if tracker.ShouldSkip(accountID, model) {
		t.Error("Should not skip before any 429")
	}

	// After 429: should skip during backoff
	tracker.Record429(accountID, model)
	if !tracker.ShouldSkip(accountID, model) {
		t.Error("Should skip immediately after 429")
	}

	// Different model: should not skip
	if tracker.ShouldSkip(accountID, "different-model") {
		t.Error("Should not skip for different model")
	}

	// Different account: should not skip
	if tracker.ShouldSkip(int64(2), model) {
		t.Error("Should not skip for different account")
	}
}

func TestAccount429Tracker_ShouldSkipAt(t *testing.T) {
	tracker := NewAccount429Tracker(&Account429TrackerConfig{
		BackoffBaseSeconds:   5,
		BackoffMaxSeconds:    300,
		BackoffMaxExponent:   6,
		HistoryWindowMinutes: 5,
	})

	accountID := int64(1)
	model := "claude-3-opus"
	now := time.Now()

	tracker.Record429(accountID, model)

	// Immediately after: should skip
	if !tracker.ShouldSkipAt(accountID, model, now) {
		t.Error("Should skip immediately after 429")
	}

	// After backoff period: should not skip
	future := now.Add(10 * time.Second)
	if tracker.ShouldSkipAt(accountID, model, future) {
		t.Error("Should not skip after backoff period")
	}
}

func TestAccount429Tracker_GetRecent429Count(t *testing.T) {
	tracker := NewAccount429Tracker(&Account429TrackerConfig{
		BackoffBaseSeconds:   1,
		BackoffMaxSeconds:    60,
		BackoffMaxExponent:   6,
		HistoryWindowMinutes: 1, // 1 minute window for faster tests
	})

	accountID := int64(1)
	model := "claude-3-opus"

	// Initially: 0 count
	if count := tracker.GetRecent429Count(accountID, model); count != 0 {
		t.Errorf("Expected 0 count initially, got %d", count)
	}

	// After 3 429s: count should be 3
	tracker.Record429(accountID, model)
	tracker.Record429(accountID, model)
	tracker.Record429(accountID, model)

	if count := tracker.GetRecent429Count(accountID, model); count != 3 {
		t.Errorf("Expected 3 count, got %d", count)
	}
}

func TestAccount429Tracker_ConcurrentAccess(t *testing.T) {
	tracker := NewAccount429Tracker(&Account429TrackerConfig{
		BackoffBaseSeconds:   1,
		BackoffMaxSeconds:    60,
		BackoffMaxExponent:   6,
		HistoryWindowMinutes: 5,
	})

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 10

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(accountID int64) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				tracker.Record429(accountID, "model")
			}
		}(int64(i % 10))
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(accountID int64) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = tracker.ShouldSkip(accountID, "model")
				_ = tracker.GetRecent429Count(accountID, "model")
			}
		}(int64(i % 10))
	}

	wg.Wait()

	// Verify no data corruption
	stats := tracker.GetStats()
	if stats["tracked_combinations"] == nil {
		t.Error("Expected stats to be available")
	}
}

func TestAccount429Tracker_ClearOldRecords(t *testing.T) {
	tracker := NewAccount429Tracker(&Account429TrackerConfig{
		BackoffBaseSeconds:   1,
		BackoffMaxSeconds:    60,
		BackoffMaxExponent:   6,
		HistoryWindowMinutes: 1,
	})

	accountID := int64(1)
	model := "claude-3-opus"

	// Record some 429s
	tracker.Record429(accountID, model)
	tracker.Record429(accountID, model)

	// Clear old records (should not affect recent records)
	tracker.ClearOldRecords()

	count := tracker.GetRecent429Count(accountID, model)
	if count != 2 {
		t.Errorf("Expected 2 count after clear, got %d", count)
	}
}

func TestAccount429Tracker_BackoffCalculation(t *testing.T) {
	tracker := NewAccount429Tracker(&Account429TrackerConfig{
		BackoffBaseSeconds:   5,
		BackoffMaxSeconds:    300,
		BackoffMaxExponent:   6,
		HistoryWindowMinutes: 5,
	})

	tests := []struct {
		count    int
		expected time.Duration
	}{
		{0, 0},
		{1, 5 * time.Second},
		{2, 10 * time.Second},
		{3, 20 * time.Second},
		{4, 40 * time.Second},
		{5, 80 * time.Second},
		{6, 160 * time.Second},
		{7, 5 * time.Minute}, // Max reached
		{10, 5 * time.Minute},
	}

	for _, tt := range tests {
		backoff := tracker.calculateBackoff(tt.count)
		if backoff != tt.expected {
			t.Errorf("calculateBackoff(%d) = %v, want %v", tt.count, backoff, tt.expected)
		}
	}
}

func TestAccount429Tracker_DefaultConfig(t *testing.T) {
	// Test with nil config (should use defaults)
	tracker := NewAccount429Tracker(nil)

	if tracker.backoffBase != 5*time.Second {
		t.Errorf("Expected default backoff base 5s, got %v", tracker.backoffBase)
	}
	if tracker.backoffMax != 5*time.Minute {
		t.Errorf("Expected default backoff max 5min, got %v", tracker.backoffMax)
	}
	if tracker.backoffMaxExp != 6 {
		t.Errorf("Expected default backoff max exp 6, got %d", tracker.backoffMaxExp)
	}
	if tracker.historyWindow != 5*time.Minute {
		t.Errorf("Expected default history window 5min, got %v", tracker.historyWindow)
	}
}
