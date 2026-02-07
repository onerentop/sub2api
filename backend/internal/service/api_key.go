package service

import "time"

// API Key status constants
const (
	StatusAPIKeyActive         = "active"
	StatusAPIKeyDisabled       = "disabled"
	StatusAPIKeyQuotaExhausted = "quota_exhausted"
	StatusAPIKeyExpired        = "expired"
)

type APIKey struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	GroupID     *int64    `json:"group_id,omitempty"`
	Status      string    `json:"status"`
	IPWhitelist []string  `json:"ip_whitelist,omitempty"`
	IPBlacklist []string  `json:"ip_blacklist,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	User        *User     `json:"user,omitempty"`
	Group       *Group    `json:"group,omitempty"`

	// Quota fields
	Quota     float64    `json:"quota"`                // Quota limit in USD (0 = unlimited)
	QuotaUsed float64    `json:"quota_used"`           // Used quota amount
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // Expiration time (nil = never expires)
}

func (k *APIKey) IsActive() bool {
	return k.Status == StatusActive
}

// IsExpired checks if the API key has expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsQuotaExhausted checks if the API key quota is exhausted
func (k *APIKey) IsQuotaExhausted() bool {
	if k.Quota <= 0 {
		return false // unlimited
	}
	return k.QuotaUsed >= k.Quota
}

// GetQuotaRemaining returns remaining quota (-1 for unlimited)
func (k *APIKey) GetQuotaRemaining() float64 {
	if k.Quota <= 0 {
		return -1 // unlimited
	}
	remaining := k.Quota - k.QuotaUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetDaysUntilExpiry returns days until expiry (-1 for never expires)
func (k *APIKey) GetDaysUntilExpiry() int {
	if k.ExpiresAt == nil {
		return -1 // never expires
	}
	duration := time.Until(*k.ExpiresAt)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}
