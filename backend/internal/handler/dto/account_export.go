// Package dto provides Data Transfer Objects for HTTP handlers.
package dto

// CLIProxyAuth represents authentication data in CLIProxyAPI format.
// This structure is compatible with the JSON auth files used by CLIProxyAPI.
type CLIProxyAuth struct {
	// Type indicates the provider type (claude, gemini, antigravity)
	Type string `json:"type"`
	// AccessToken is the OAuth2 access token
	AccessToken string `json:"access_token,omitempty"`
	// RefreshToken is used to obtain new access tokens
	RefreshToken string `json:"refresh_token,omitempty"`
	// IDToken is the JWT ID token (Claude specific)
	IDToken string `json:"id_token,omitempty"`
	// Email is the account email address
	Email string `json:"email,omitempty"`
	// ProjectID is the GCP project ID (Gemini/Antigravity specific)
	ProjectID string `json:"project_id,omitempty"`
	// Expired is the token expiration time in RFC3339 format
	Expired string `json:"expired,omitempty"`
	// LastRefresh is the timestamp of last token refresh
	LastRefresh string `json:"last_refresh,omitempty"`
	// ExpiresIn is the token lifetime in seconds
	ExpiresIn int64 `json:"expires_in,omitempty"`
	// Timestamp is the Unix timestamp in milliseconds
	Timestamp int64 `json:"timestamp,omitempty"`
}

// ExportAccountsRequest represents the request for exporting accounts.
type ExportAccountsRequest struct {
	// AccountIDs specifies which accounts to export. Empty means export all OAuth accounts.
	AccountIDs []int64 `json:"account_ids"`
	// Platforms optionally filters by platform (claude, gemini, antigravity)
	Platforms []string `json:"platforms,omitempty"`
}

// ExportAccountsResponse represents the response for account export.
type ExportAccountsResponse struct {
	// Accounts contains the exported account data in CLIProxyAPI format
	Accounts []CLIProxyAuth `json:"accounts"`
	// Total is the total number of accounts processed
	Total int `json:"total"`
	// Exported is the number of accounts successfully exported
	Exported int `json:"exported"`
	// Skipped is the number of accounts skipped (non-OAuth types)
	Skipped int `json:"skipped"`
}

// ImportAccountsRequest represents the request for importing accounts.
type ImportAccountsRequest struct {
	// Accounts contains the account data to import in CLIProxyAPI format
	Accounts []CLIProxyAuth `json:"accounts" binding:"required,min=1"`
	// GroupIDs optionally assigns imported accounts to these groups
	GroupIDs []int64 `json:"group_ids,omitempty"`
	// SkipExisting skips accounts that already exist (matched by email+platform)
	SkipExisting bool `json:"skip_existing"`
}

// ImportAccountsResponse represents the response for account import.
type ImportAccountsResponse struct {
	// Created is the number of new accounts created
	Created int `json:"created"`
	// Updated is the number of existing accounts updated
	Updated int `json:"updated"`
	// Skipped is the number of accounts skipped
	Skipped int `json:"skipped"`
	// Failed is the number of accounts that failed to import
	Failed int `json:"failed"`
	// Results contains detailed results for each account
	Results []ImportResult `json:"results"`
}

// ImportResult represents the result of importing a single account.
type ImportResult struct {
	// Email is the account email (if available)
	Email string `json:"email,omitempty"`
	// Type is the platform type
	Type string `json:"type"`
	// Action indicates what happened: created, updated, skipped, failed
	Action string `json:"action"`
	// Error contains the error message if action is "failed"
	Error string `json:"error,omitempty"`
}
