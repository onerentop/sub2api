package service

import (
	"sync"
	"time"
)

// Account429Record 记录单个账户的 429 状态
type Account429Record struct {
	ResetAt    time.Time // 429 重置时间
	RecordedAt time.Time // 记录时间
	QuotaScope string    // 可选：Antigravity quota scope
}

// Account429Tracker 内存级 429 状态跟踪器
// 用于快速过滤限流账户，避免 SchedulerSnapshot 同步延迟导致的重复选中
type Account429Tracker struct {
	mu      sync.RWMutex
	records map[int64]*Account429Record // accountID -> record
}

// NewAccount429Tracker 创建 429 跟踪器
func NewAccount429Tracker() *Account429Tracker {
	return &Account429Tracker{
		records: make(map[int64]*Account429Record),
	}
}

// Record429 记录账户 429 状态
func (t *Account429Tracker) Record429(accountID int64, resetAt time.Time, quotaScope string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.records[accountID] = &Account429Record{
		ResetAt:    resetAt,
		RecordedAt: time.Now(),
		QuotaScope: quotaScope,
	}
}

// IsAccountAvailable 检查账户是否可用（未被 429 限流）
func (t *Account429Tracker) IsAccountAvailable(accountID int64) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	record, exists := t.records[accountID]
	if !exists {
		return true
	}

	// 如果已过重置时间，账户可用
	if time.Now().After(record.ResetAt) {
		return true
	}

	return false
}

// IsAccountAvailableForScope 检查账户在特定 quota scope 下是否可用
// 用于 Antigravity 平台的细粒度限流控制
func (t *Account429Tracker) IsAccountAvailableForScope(accountID int64, quotaScope string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	record, exists := t.records[accountID]
	if !exists {
		return true
	}

	// 如果 scope 不匹配，账户对当前 scope 可用
	if record.QuotaScope != "" && record.QuotaScope != quotaScope {
		return true
	}

	// 如果已过重置时间，账户可用
	if time.Now().After(record.ResetAt) {
		return true
	}

	return false
}

// ClearExpired 清理已过期的记录（可定期调用或惰性清理）
func (t *Account429Tracker) ClearExpired() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	cleared := 0
	for id, record := range t.records {
		if now.After(record.ResetAt) {
			delete(t.records, id)
			cleared++
		}
	}
	return cleared
}

// GetResetTime 获取账户的 429 重置时间（如果存在）
func (t *Account429Tracker) GetResetTime(accountID int64) *time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()

	record, exists := t.records[accountID]
	if !exists {
		return nil
	}

	if time.Now().After(record.ResetAt) {
		return nil
	}

	return &record.ResetAt
}
