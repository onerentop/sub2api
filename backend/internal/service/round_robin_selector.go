package service

import (
	"fmt"
	"sync"
)

// RoundRobinSelector provides round-robin account selection
// to distribute requests evenly across accounts and avoid single account burnout
type RoundRobinSelector struct {
	mu      sync.Mutex
	cursors map[string]int // key -> current cursor position
}

// NewRoundRobinSelector creates a new round-robin selector
func NewRoundRobinSelector() *RoundRobinSelector {
	return &RoundRobinSelector{
		cursors: make(map[string]int),
	}
}

// buildKey constructs a unique key for the (groupID, platform, model) combination
func (s *RoundRobinSelector) buildKey(groupID *int64, platform, model string) string {
	if groupID != nil {
		return fmt.Sprintf("g%d:%s:%s", *groupID, platform, model)
	}
	return fmt.Sprintf("g0:%s:%s", platform, model)
}

// Select returns the next index in round-robin order for the given context
// candidateCount is the total number of available candidates
// Returns an index in [0, candidateCount)
func (s *RoundRobinSelector) Select(groupID *int64, platform, model string, candidateCount int) int {
	if candidateCount <= 0 {
		return 0
	}
	if candidateCount == 1 {
		return 0
	}

	key := s.buildKey(groupID, platform, model)

	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.cursors[key]

	// Prevent integer overflow (reset at a safe threshold)
	if index >= 2_147_483_640 {
		index = 0
	}

	s.cursors[key] = index + 1

	return index % candidateCount
}

// Reset clears all cursor positions
func (s *RoundRobinSelector) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cursors = make(map[string]int)
}

// GetStats returns current selector statistics for monitoring
func (s *RoundRobinSelector) GetStats() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]any{
		"active_keys": len(s.cursors),
	}
}
