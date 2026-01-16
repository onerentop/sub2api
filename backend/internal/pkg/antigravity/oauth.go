package antigravity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// Google OAuth 端点
	AuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	TokenURL     = "https://oauth2.googleapis.com/token"
	UserInfoURL  = "https://www.googleapis.com/oauth2/v2/userinfo"

	// Antigravity OAuth 客户端凭证
	ClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	ClientSecret = "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"

	// 固定的 redirect_uri（用户需手动复制 code）
	RedirectURI = "http://localhost:8085/callback"

	// OAuth scopes
	Scopes = "https://www.googleapis.com/auth/cloud-platform " +
		"https://www.googleapis.com/auth/userinfo.email " +
		"https://www.googleapis.com/auth/userinfo.profile " +
		"https://www.googleapis.com/auth/cclog " +
		"https://www.googleapis.com/auth/experimentsandconfigs"

	// User-Agent（与 CLIProxyAPI/Antigravity CLI 保持一致）
	// 格式：antigravity/{version} {os}/{arch}
	UserAgent = "antigravity/1.104.0 linux/amd64"

	// Session 过期时间
	SessionTTL = 30 * time.Minute

	// WakeSessionTTL 唤醒 session 过期时间（OAuth 完成后延长有效期）
	WakeSessionTTL = 24 * time.Hour

	// URL 可用性 TTL（不可用 URL 的恢复时间）
	URLAvailabilityTTL = 5 * time.Minute
)

// BaseURLs 定义 Antigravity API 端点，按优先级排序
// 参考 CLIProxyAPI: sandbox-daily → daily → prod（sandbox 限流更宽松，优先使用）
var BaseURLs = []string{
	"https://daily-cloudcode-pa.sandbox.googleapis.com", // sandbox-daily（优先！限流最宽松）
	"https://daily-cloudcode-pa.googleapis.com",         // daily（中间）
	"https://cloudcode-pa.googleapis.com",               // prod（最后！限流最严格）
}

// BaseURL 默认 URL（保持向后兼容，使用 sandbox-daily）
var BaseURL = BaseURLs[0]

// URLAvailability 管理 URL 可用性状态（带 TTL 自动恢复）
type URLAvailability struct {
	mu          sync.RWMutex
	unavailable map[string]time.Time // URL -> 恢复时间
	ttl         time.Duration
}

// DefaultURLAvailability 全局 URL 可用性管理器
var DefaultURLAvailability = NewURLAvailability(URLAvailabilityTTL)

// NewURLAvailability 创建 URL 可用性管理器
func NewURLAvailability(ttl time.Duration) *URLAvailability {
	return &URLAvailability{
		unavailable: make(map[string]time.Time),
		ttl:         ttl,
	}
}

// MarkUnavailable 标记 URL 临时不可用
func (u *URLAvailability) MarkUnavailable(url string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.unavailable[url] = time.Now().Add(u.ttl)
}

// IsAvailable 检查 URL 是否可用
func (u *URLAvailability) IsAvailable(url string) bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	expiry, exists := u.unavailable[url]
	if !exists {
		return true
	}
	return time.Now().After(expiry)
}

// GetAvailableURLs 返回可用的 URL 列表（保持优先级顺序）
// 注意：参考 CLIProxyAPI，每个请求都独立尝试所有 URL，不再过滤不可用的 URL
// 这样可以避免所有 URL 被标记为不可用后无法恢复的问题
func (u *URLAvailability) GetAvailableURLs() []string {
	// 直接返回所有 URL，不进行可用性过滤
	// CLIProxyAPI 的做法是每个请求都独立尝试所有 URL
	return BaseURLs
}

// OAuthSession 保存 OAuth 授权流程的临时状态
type OAuthSession struct {
	State        string    `json:"state"`
	CodeVerifier string    `json:"code_verifier"`
	ProxyURL     string    `json:"proxy_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`

	// Token 信息（OAuth 完成后填充，用于后续唤醒请求）
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
	Email        string `json:"email,omitempty"`
	ProjectID    string `json:"project_id,omitempty"`
}

// HasToken 检查 session 是否已完成 OAuth 并包含 token
func (s *OAuthSession) HasToken() bool {
	return s.AccessToken != ""
}

// SessionStore OAuth session 存储
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*OAuthSession
	stopCh   chan struct{}
}

func NewSessionStore() *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*OAuthSession),
		stopCh:   make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (s *SessionStore) Set(sessionID string, session *OAuthSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = session
}

func (s *SessionStore) Get(sessionID string) (*OAuthSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}
	if time.Since(session.CreatedAt) > SessionTTL {
		return nil, false
	}
	return session, true
}

func (s *SessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// Update 更新 session 并延长有效期（用于 OAuth 完成后存储 token）
func (s *SessionStore) Update(sessionID string, session *OAuthSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 重置创建时间以延长有效期
	session.CreatedAt = time.Now()
	s.sessions[sessionID] = session
}

// GetWithWakeTTL 获取 session，使用唤醒 TTL 判断过期
func (s *SessionStore) GetWithWakeTTL(sessionID string) (*OAuthSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}
	// 对于已完成 OAuth 的 session，使用更长的 TTL
	ttl := SessionTTL
	if session.HasToken() {
		ttl = WakeSessionTTL
	}
	if time.Since(session.CreatedAt) > ttl {
		return nil, false
	}
	return session, true
}

func (s *SessionStore) Stop() {
	select {
	case <-s.stopCh:
		return
	default:
		close(s.stopCh)
	}
}

func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			for id, session := range s.sessions {
				// 对于已完成 OAuth 的 session，使用更长的 TTL
				ttl := SessionTTL
				if session.HasToken() {
					ttl = WakeSessionTTL
				}
				if time.Since(session.CreatedAt) > ttl {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func GenerateState() (string, error) {
	bytes, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return base64URLEncode(bytes), nil
}

func GenerateSessionID() (string, error) {
	bytes, err := GenerateRandomBytes(16)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateCodeVerifier() (string, error) {
	bytes, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return base64URLEncode(bytes), nil
}

func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64URLEncode(hash[:])
}

func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// BuildAuthorizationURL 构建 Google OAuth 授权 URL
func BuildAuthorizationURL(state, codeChallenge string) string {
	params := url.Values{}
	params.Set("client_id", ClientID)
	params.Set("redirect_uri", RedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", Scopes)
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	params.Set("include_granted_scopes", "true")

	return fmt.Sprintf("%s?%s", AuthorizeURL, params.Encode())
}

// GenerateMockProjectID 生成随机 project_id（当 API 不返回时使用）
// 格式：{形容词}-{名词}-{5位随机字符}
func GenerateMockProjectID() string {
	adjectives := []string{"useful", "bright", "swift", "calm", "bold"}
	nouns := []string{"fuze", "wave", "spark", "flow", "core"}

	randBytes, _ := GenerateRandomBytes(7)

	adj := adjectives[int(randBytes[0])%len(adjectives)]
	noun := nouns[int(randBytes[1])%len(nouns)]

	// 生成 5 位随机字符（a-z0-9）
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 5)
	for i := 0; i < 5; i++ {
		suffix[i] = charset[int(randBytes[i+2])%len(charset)]
	}

	return fmt.Sprintf("%s-%s-%s", adj, noun, string(suffix))
}
