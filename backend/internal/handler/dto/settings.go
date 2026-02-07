package dto

// SystemSettings represents the admin settings API response payload.
type SystemSettings struct {
	RegistrationEnabled         bool     `json:"registration_enabled"`
	EmailVerifyEnabled          bool     `json:"email_verify_enabled"`
	PromoCodeEnabled            bool     `json:"promo_code_enabled"`
	PasswordResetEnabled        bool     `json:"password_reset_enabled"`
	InvitationCodeEnabled       bool     `json:"invitation_code_enabled"`
	TotpEnabled                 bool     `json:"totp_enabled"`                   // TOTP 双因素认证
	TotpEncryptionKeyConfigured bool     `json:"totp_encryption_key_configured"` // TOTP 加密密钥是否已配置
	EmailDomainWhitelist        []string `json:"email_domain_whitelist"`         // 允许注册的邮箱后缀

	SMTPHost               string `json:"smtp_host"`
	SMTPPort               int    `json:"smtp_port"`
	SMTPUsername           string `json:"smtp_username"`
	SMTPPasswordConfigured bool   `json:"smtp_password_configured"`
	SMTPFrom               string `json:"smtp_from_email"`
	SMTPFromName           string `json:"smtp_from_name"`
	SMTPUseTLS             bool   `json:"smtp_use_tls"`

	TurnstileEnabled             bool   `json:"turnstile_enabled"`
	TurnstileSiteKey             string `json:"turnstile_site_key"`
	TurnstileSecretKeyConfigured bool   `json:"turnstile_secret_key_configured"`

	LinuxDoConnectEnabled                bool   `json:"linuxdo_connect_enabled"`
	LinuxDoConnectClientID               string `json:"linuxdo_connect_client_id"`
	LinuxDoConnectClientSecretConfigured bool   `json:"linuxdo_connect_client_secret_configured"`
	LinuxDoConnectRedirectURL            string `json:"linuxdo_connect_redirect_url"`

	SiteName                    string `json:"site_name"`
	SiteLogo                    string `json:"site_logo"`
	SiteSubtitle                string `json:"site_subtitle"`
	APIBaseURL                  string `json:"api_base_url"`
	ContactInfo                 string `json:"contact_info"`
	DocURL                      string `json:"doc_url"`
	HomeContent                 string `json:"home_content"`
	HideCcsImportButton         bool   `json:"hide_ccs_import_button"`
	PurchaseSubscriptionEnabled bool   `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL     string `json:"purchase_subscription_url"`

	DefaultConcurrency int     `json:"default_concurrency"`
	DefaultBalance     float64 `json:"default_balance"`

	// Model fallback configuration
	EnableModelFallback      bool   `json:"enable_model_fallback"`
	FallbackModelAnthropic   string `json:"fallback_model_anthropic"`
	FallbackModelOpenAI      string `json:"fallback_model_openai"`
	FallbackModelGemini      string `json:"fallback_model_gemini"`
	FallbackModelAntigravity string `json:"fallback_model_antigravity"`

	// Identity patch configuration (Claude -> Gemini)
	EnableIdentityPatch bool   `json:"enable_identity_patch"`
	IdentityPatchPrompt string `json:"identity_patch_prompt"`

	// Ops monitoring (vNext)
	OpsMonitoringEnabled         bool   `json:"ops_monitoring_enabled"`
	OpsRealtimeMonitoringEnabled bool   `json:"ops_realtime_monitoring_enabled"`
	OpsQueryModeDefault          string `json:"ops_query_mode_default"`
	OpsMetricsIntervalSeconds    int    `json:"ops_metrics_interval_seconds"`

	// Payment settings (YiPay)
	PaymentEnabled            bool    `json:"payment_enabled"`
	PaymentYiPayAPIURL        string  `json:"payment_yipay_api_url"`
	PaymentYiPayPID           string  `json:"payment_yipay_pid"`
	PaymentYiPayKeyConfigured bool    `json:"payment_yipay_key_configured"`
	PaymentYiPayNotifyURL     string  `json:"payment_yipay_notify_url"`
	PaymentYiPayReturnURL     string  `json:"payment_yipay_return_url"`
	PaymentMinAmount          float64 `json:"payment_min_amount"`
	PaymentMaxAmount          float64 `json:"payment_max_amount"`
	PaymentAuditThreshold     float64 `json:"payment_audit_threshold"`
	PaymentCNYToValueRate     float64 `json:"payment_cny_to_value_rate"`
}

type PublicSettings struct {
	RegistrationEnabled         bool   `json:"registration_enabled"`
	EmailVerifyEnabled          bool   `json:"email_verify_enabled"`
	PromoCodeEnabled            bool   `json:"promo_code_enabled"`
	PasswordResetEnabled        bool   `json:"password_reset_enabled"`
	InvitationCodeEnabled       bool   `json:"invitation_code_enabled"`
	TotpEnabled                 bool   `json:"totp_enabled"` // TOTP 双因素认证
	EmailDomainWhitelistEnabled bool   `json:"email_domain_whitelist_enabled"`
	TurnstileEnabled            bool   `json:"turnstile_enabled"`
	TurnstileSiteKey            string `json:"turnstile_site_key"`
	SiteName                    string `json:"site_name"`
	SiteLogo                    string `json:"site_logo"`
	SiteSubtitle                string `json:"site_subtitle"`
	APIBaseURL                  string `json:"api_base_url"`
	ContactInfo                 string `json:"contact_info"`
	DocURL                      string `json:"doc_url"`
	HomeContent                 string `json:"home_content"`
	HideCcsImportButton         bool   `json:"hide_ccs_import_button"`
	PurchaseSubscriptionEnabled bool   `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL     string `json:"purchase_subscription_url"`
	LinuxDoOAuthEnabled         bool   `json:"linuxdo_oauth_enabled"`
	Version                     string `json:"version"`
}

// StreamTimeoutSettings 流超时处理配置 DTO
type StreamTimeoutSettings struct {
	Enabled                bool   `json:"enabled"`
	Action                 string `json:"action"`
	TempUnschedMinutes     int    `json:"temp_unsched_minutes"`
	ThresholdCount         int    `json:"threshold_count"`
	ThresholdWindowMinutes int    `json:"threshold_window_minutes"`
}
