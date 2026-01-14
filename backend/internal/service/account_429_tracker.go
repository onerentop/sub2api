package service

import (
	"fmt"
	"sync"
	"time"
)

// Account429Tracker 追踪账户的 429 错误历史，实现指数退避调度
// 线程安全，支持按 accountID:model 粒度追踪
type Account429Tracker struct {
	mu       sync.RWMutex
	counters map[string]*ModelCounter // key: "accountID:model"

	// 配置参数
	backoffBase   time.Duration // 基础退避时间，默认 5s
	backoffMax    time.Duration // 最大退避时间，默认 5min
	backoffMaxExp int           // 最大指数，默认 6
	historyWindow time.Duration // 历史窗口，默认 5min
}

// ModelCounter 记录单个 accountID:model 组合的 429 历史
type ModelCounter struct {
	window       []time.Time // 滑动窗口记录 429 发生时间
	backoffUntil time.Time   // 退避结束时间
}

// Account429TrackerConfig 追踪器配置
type Account429TrackerConfig struct {
	BackoffBaseSeconds   int // 基础退避时间（秒），默认 5
	BackoffMaxSeconds    int // 最大退避时间（秒），默认 300
	BackoffMaxExponent   int // 最大指数，默认 6
	HistoryWindowMinutes int // 历史窗口（分钟），默认 5
}

// NewAccount429Tracker 创建新的 429 追踪器
func NewAccount429Tracker(cfg *Account429TrackerConfig) *Account429Tracker {
	t := &Account429Tracker{
		counters:      make(map[string]*ModelCounter),
		backoffBase:   5 * time.Second,
		backoffMax:    5 * time.Minute,
		backoffMaxExp: 6,
		historyWindow: 5 * time.Minute,
	}

	if cfg != nil {
		if cfg.BackoffBaseSeconds > 0 {
			t.backoffBase = time.Duration(cfg.BackoffBaseSeconds) * time.Second
		}
		if cfg.BackoffMaxSeconds > 0 {
			t.backoffMax = time.Duration(cfg.BackoffMaxSeconds) * time.Second
		}
		if cfg.BackoffMaxExponent > 0 {
			t.backoffMaxExp = cfg.BackoffMaxExponent
		}
		if cfg.HistoryWindowMinutes > 0 {
			t.historyWindow = time.Duration(cfg.HistoryWindowMinutes) * time.Minute
		}
	}

	return t
}

// trackerKey 生成追踪器 key
func trackerKey(accountID int64, model string) string {
	return fmt.Sprintf("%d:%s", accountID, model)
}

// Record429 记录一次 429 事件，计算指数退避时间
func (t *Account429Tracker) Record429(accountID int64, model string) {
	key := trackerKey(accountID, model)
	now := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	counter := t.getOrCreateLocked(key)

	// 添加到滑动窗口
	counter.window = append(counter.window, now)

	// 清理过期记录
	counter.window = t.cleanWindowLocked(counter.window, now)

	// 计算指数退避
	count := len(counter.window)
	backoff := t.calculateBackoff(count)
	counter.backoffUntil = now.Add(backoff)
}

// ShouldSkip 检查账户+模型是否处于退避状态，应跳过选择
func (t *Account429Tracker) ShouldSkip(accountID int64, model string) bool {
	return t.ShouldSkipAt(accountID, model, time.Now())
}

// ShouldSkipAt 检查账户+模型在指定时间是否处于退避状态
func (t *Account429Tracker) ShouldSkipAt(accountID int64, model string, now time.Time) bool {
	key := trackerKey(accountID, model)

	t.mu.RLock()
	defer t.mu.RUnlock()

	counter, exists := t.counters[key]
	if !exists {
		return false
	}

	return now.Before(counter.backoffUntil)
}

// GetBackoffUntil 获取退避结束时间
func (t *Account429Tracker) GetBackoffUntil(accountID int64, model string) time.Time {
	key := trackerKey(accountID, model)

	t.mu.RLock()
	defer t.mu.RUnlock()

	counter, exists := t.counters[key]
	if !exists {
		return time.Time{}
	}

	return counter.backoffUntil
}

// GetRecent429Count 获取最近历史窗口内的 429 次数
func (t *Account429Tracker) GetRecent429Count(accountID int64, model string) int {
	return t.GetRecent429CountAt(accountID, model, time.Now())
}

// GetRecent429CountAt 获取指定时间点之前历史窗口内的 429 次数
func (t *Account429Tracker) GetRecent429CountAt(accountID int64, model string, now time.Time) int {
	key := trackerKey(accountID, model)

	t.mu.RLock()
	defer t.mu.RUnlock()

	counter, exists := t.counters[key]
	if !exists {
		return 0
	}

	// 计算窗口内的有效记录数
	cutoff := now.Add(-t.historyWindow)
	count := 0
	for _, ts := range counter.window {
		if ts.After(cutoff) {
			count++
		}
	}

	return count
}

// ClearOldRecords 清理所有过期记录（可由后台任务调用）
func (t *Account429Tracker) ClearOldRecords() {
	now := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	for key, counter := range t.counters {
		// 清理滑动窗口
		counter.window = t.cleanWindowLocked(counter.window, now)

		// 如果窗口为空且退避已结束，删除整个 counter
		if len(counter.window) == 0 && now.After(counter.backoffUntil) {
			delete(t.counters, key)
		}
	}
}

// RecordSuccess 记录一次成功请求，可选择性地减少退避惩罚
// 当前实现：成功不影响历史记录，仅依赖时间窗口自然过期
func (t *Account429Tracker) RecordSuccess(accountID int64, model string) {
	// 当前设计：成功请求不主动清除 429 记录
	// 历史记录通过滑动窗口自然过期
	// 未来可扩展：连续成功后加速退避恢复
}

// getOrCreateLocked 获取或创建 counter（调用者需持有写锁）
func (t *Account429Tracker) getOrCreateLocked(key string) *ModelCounter {
	counter, exists := t.counters[key]
	if !exists {
		counter = &ModelCounter{
			window: make([]time.Time, 0, 8),
		}
		t.counters[key] = counter
	}
	return counter
}

// cleanWindowLocked 清理滑动窗口中的过期记录
func (t *Account429Tracker) cleanWindowLocked(window []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-t.historyWindow)

	// 找到第一个未过期的索引
	firstValid := 0
	for i, ts := range window {
		if ts.After(cutoff) {
			firstValid = i
			break
		}
		if i == len(window)-1 {
			// 全部过期
			return window[:0]
		}
	}

	if firstValid > 0 {
		// 移除过期记录
		copy(window, window[firstValid:])
		window = window[:len(window)-firstValid]
	}

	return window
}

// calculateBackoff 计算指数退避时间
// 退避公式: base * 2^(count-1)，上限为 max
func (t *Account429Tracker) calculateBackoff(count int) time.Duration {
	if count <= 0 {
		return 0
	}

	exp := count - 1
	if exp > t.backoffMaxExp {
		exp = t.backoffMaxExp
	}

	backoff := t.backoffBase * time.Duration(1<<exp)
	if backoff > t.backoffMax {
		backoff = t.backoffMax
	}

	return backoff
}

// GetStats returns tracker statistics for monitoring/debugging
func (t *Account429Tracker) GetStats() map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	activeBackoffs := 0
	totalRecords := 0

	for _, counter := range t.counters {
		if now.Before(counter.backoffUntil) {
			activeBackoffs++
		}
		totalRecords += len(counter.window)
	}

	return map[string]any{
		"tracked_combinations": len(t.counters),
		"active_backoffs":      activeBackoffs,
		"total_429_records":    totalRecords,
		"history_window":       t.historyWindow.String(),
		"backoff_base":         t.backoffBase.String(),
		"backoff_max":          t.backoffMax.String(),
	}
}
