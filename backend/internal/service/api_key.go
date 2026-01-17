package service

import "time"

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
}

func (k *APIKey) IsActive() bool {
	return k.Status == StatusActive
}
