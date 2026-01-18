package service

import (
	"fmt"
	"time"
)

type Proxy struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Protocol  string    `json:"protocol"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Username  string    `json:"username,omitempty"`
	Password  string    `json:"password,omitempty"` // Internal use only - filtered out in API responses
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (p *Proxy) IsActive() bool {
	return p.Status == StatusActive
}

func (p *Proxy) URL() string {
	if p.Username != "" && p.Password != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%d", p.Protocol, p.Username, p.Password, p.Host, p.Port)
	}
	return fmt.Sprintf("%s://%s:%d", p.Protocol, p.Host, p.Port)
}

type ProxyWithAccountCount struct {
	Proxy
	AccountCount   int64  `json:"account_count"`
	LatencyMs      *int64 `json:"latency_ms,omitempty"`
	LatencyStatus  string `json:"latency_status,omitempty"`
	LatencyMessage string `json:"latency_message,omitempty"`
	IPAddress      string `json:"ip_address,omitempty"`
	Country        string `json:"country,omitempty"`
	CountryCode    string `json:"country_code,omitempty"`
	Region         string `json:"region,omitempty"`
	City           string `json:"city,omitempty"`
}

type ProxyAccountSummary struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Platform string  `json:"platform"`
	Type     string  `json:"type"`
	Notes    *string `json:"notes,omitempty"`
}
