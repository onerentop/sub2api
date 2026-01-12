package service

type SystemSettings struct {
	RegistrationEnabled  bool
	EmailVerifyEnabled   bool
	EmailDomainWhitelist []string // 邮箱域名白名单

	SMTPHost               string
	SMTPPort               int
	SMTPUsername           string
	SMTPPassword           string
	SMTPPasswordConfigured bool
	SMTPFrom               string
	SMTPFromName           string
	SMTPUseTLS             bool

	TurnstileEnabled             bool
	TurnstileSiteKey             string
	TurnstileSecretKey           string
	TurnstileSecretKeyConfigured bool

	// LinuxDo Connect OAuth 登录（终端用户 SSO）
	LinuxDoConnectEnabled                bool
	LinuxDoConnectClientID               string
	LinuxDoConnectClientSecret           string
	LinuxDoConnectClientSecretConfigured bool
	LinuxDoConnectRedirectURL            string

	SiteName     string
	SiteLogo     string
	SiteSubtitle string
	APIBaseURL   string
	ContactInfo  string
	DocURL       string
	HomeContent  string

	DefaultConcurrency int
	DefaultBalance     float64

	// Model fallback configuration
	EnableModelFallback      bool   `json:"enable_model_fallback"`
	FallbackModelAnthropic   string `json:"fallback_model_anthropic"`
	FallbackModelOpenAI      string `json:"fallback_model_openai"`
	FallbackModelGemini      string `json:"fallback_model_gemini"`
	FallbackModelAntigravity string `json:"fallback_model_antigravity"`

	// Identity patch configuration (Claude -> Gemini)
	EnableIdentityPatch bool   `json:"enable_identity_patch"`
	IdentityPatchPrompt string `json:"identity_patch_prompt"`
}

type PublicSettings struct {
	RegistrationEnabled         bool
	EmailVerifyEnabled          bool
	EmailDomainWhitelistEnabled bool // 是否启用邮箱域名白名单
	TurnstileEnabled            bool
	TurnstileSiteKey            string
	SiteName                    string
	SiteLogo                    string
	SiteSubtitle                string
	APIBaseURL                  string
	ContactInfo                 string
	DocURL                      string
	HomeContent                 string
	LinuxDoOAuthEnabled         bool
	Version                     string
}
