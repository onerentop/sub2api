package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
)

// OAuthProviderStrategy 定义各提供商的 OAuth 策略接口
type OAuthProviderStrategy interface {
	// Name 返回提供商名称
	Name() string
	// BuildAuthURL 构建授权 URL
	BuildAuthURL(state, redirectURI string, provider *OAuthProvider) string
	// ExchangeCode 用授权码换取用户信息
	ExchangeCode(ctx context.Context, code, redirectURI string, provider *OAuthProvider) (*OAuthUserInfo, error)
}

// SocialOAuthService 社交登录服务
type SocialOAuthService struct {
	providerRepo OAuthProviderRepository
	bindingRepo  UserOAuthBindingRepository
	userRepo     UserRepository
	authService  *AuthService
	sessionStore *oauth.SessionStore
	strategies   map[string]OAuthProviderStrategy
	mu           sync.RWMutex
}

// NewSocialOAuthService 创建社交登录服务
func NewSocialOAuthService(
	providerRepo OAuthProviderRepository,
	bindingRepo UserOAuthBindingRepository,
	userRepo UserRepository,
	authService *AuthService,
) *SocialOAuthService {
	svc := &SocialOAuthService{
		providerRepo: providerRepo,
		bindingRepo:  bindingRepo,
		userRepo:     userRepo,
		authService:  authService,
		sessionStore: oauth.NewSessionStore(),
		strategies:   make(map[string]OAuthProviderStrategy),
	}

	// 注册策略
	svc.RegisterStrategy(NewGoogleOAuthStrategy())
	svc.RegisterStrategy(NewGitHubOAuthStrategy())

	return svc
}

// RegisterStrategy 注册 OAuth 策略
func (s *SocialOAuthService) RegisterStrategy(strategy OAuthProviderStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.strategies[strategy.Name()] = strategy
}

// GetStrategy 获取 OAuth 策略
func (s *SocialOAuthService) GetStrategy(name string) (OAuthProviderStrategy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	strategy, ok := s.strategies[name]
	return strategy, ok
}

// GetEnabledProviders 获取已启用的提供商列表（公开 API）
func (s *SocialOAuthService) GetEnabledProviders(ctx context.Context) ([]OAuthProviderPublicInfo, error) {
	providers, err := s.providerRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]OAuthProviderPublicInfo, 0, len(providers))
	for _, p := range providers {
		// 只返回有策略实现的提供商
		if _, ok := s.GetStrategy(p.Name); ok {
			result = append(result, p.ToPublicInfo())
		}
	}
	return result, nil
}

// StartOAuthResult 启动 OAuth 流程结果
type StartOAuthResult struct {
	AuthURL   string `json:"auth_url"`
	SessionID string `json:"session_id"`
}

// StartOAuth 启动 OAuth 登录流程
func (s *SocialOAuthService) StartOAuth(ctx context.Context, providerName, redirectTo, callbackURL string) (*StartOAuthResult, error) {
	// 获取提供商配置
	provider, err := s.providerRepo.GetByName(ctx, providerName)
	if err != nil {
		return nil, err
	}
	if !provider.Enabled {
		return nil, ErrOAuthProviderDisabled
	}
	if provider.ClientID == "" {
		return nil, ErrOAuthProviderDisabled
	}

	// 获取策略
	strategy, ok := s.GetStrategy(providerName)
	if !ok {
		return nil, ErrOAuthProviderNotFound
	}

	// 生成 state 和 session
	state, err := oauth.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	sessionID, err := oauth.GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}

	// 存储 session
	session := &oauth.OAuthSession{
		State:     state,
		Scope:     provider.GetConfigString("scopes"),
		CreatedAt: time.Now(),
	}
	// 存储 redirectTo 以便回调后跳转
	if redirectTo != "" {
		session.ProxyURL = redirectTo // 复用 ProxyURL 字段存储 redirectTo
	}
	s.sessionStore.Set(sessionID, session)

	// 构建授权 URL
	authURL := strategy.BuildAuthURL(state, callbackURL, provider)

	return &StartOAuthResult{
		AuthURL:   authURL,
		SessionID: sessionID,
	}, nil
}

// HandleCallbackResult 处理回调结果
type HandleCallbackResult struct {
	Token      string `json:"token"`
	RedirectTo string `json:"redirect_to"`
	IsNewUser  bool   `json:"is_new_user"`
}

// HandleCallback 处理 OAuth 回调，登录或注册用户
func (s *SocialOAuthService) HandleCallback(ctx context.Context, providerName, code, state, sessionID, callbackURL string) (*HandleCallbackResult, error) {
	// 验证 session
	session, ok := s.sessionStore.Get(sessionID)
	if !ok {
		return nil, ErrOAuthStateMismatch
	}
	defer s.sessionStore.Delete(sessionID)

	if session.State != state {
		return nil, ErrOAuthStateMismatch
	}

	// 获取提供商配置
	provider, err := s.providerRepo.GetByName(ctx, providerName)
	if err != nil {
		return nil, err
	}
	if !provider.Enabled {
		return nil, ErrOAuthProviderDisabled
	}

	// 获取策略并换取用户信息
	strategy, ok := s.GetStrategy(providerName)
	if !ok {
		return nil, ErrOAuthProviderNotFound
	}

	userInfo, err := strategy.ExchangeCode(ctx, code, callbackURL, provider)
	if err != nil {
		log.Printf("[SocialOAuth] exchange code failed for %s: %v", providerName, err)
		return nil, ErrOAuthCodeExchangeFailed
	}

	// 查找是否已有绑定
	binding, err := s.bindingRepo.GetByProviderUserID(ctx, providerName, userInfo.ProviderUserID)
	if err != nil {
		return nil, err
	}

	var user *User
	isNewUser := false

	if binding != nil {
		// 已有绑定，直接登录
		user, err = s.userRepo.GetByID(ctx, binding.UserID)
		if err != nil {
			return nil, err
		}
		if !user.IsActive() {
			return nil, ErrUserNotActive
		}
	} else {
		// 无绑定，尝试用邮箱查找或创建用户
		email := userInfo.Email
		if email == "" {
			// 生成合成邮箱
			email = fmt.Sprintf("%s-%s@oauth.local", providerName, userInfo.ProviderUserID)
		}

		// 使用 AuthService 的 LoginOrRegisterOAuth
		_, user, err = s.authService.LoginOrRegisterOAuth(ctx, email, userInfo.Username)
		if err != nil {
			return nil, err
		}
		isNewUser = true

		// 创建绑定
		newBinding := &UserOAuthBinding{
			UserID:           user.ID,
			Provider:         providerName,
			ProviderUserID:   userInfo.ProviderUserID,
			ProviderEmail:    stringPtr(userInfo.Email),
			ProviderUsername: stringPtr(userInfo.Username),
			ProviderAvatar:   stringPtr(userInfo.Avatar),
			AccessToken:      stringPtr(userInfo.AccessToken),
			RefreshToken:     stringPtr(userInfo.RefreshToken),
		}
		if err := s.bindingRepo.Create(ctx, newBinding); err != nil {
			log.Printf("[SocialOAuth] create binding failed: %v", err)
			// 不影响登录，只记录日志
		}
	}

	// 生成 JWT
	token, err := s.authService.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	redirectTo := session.ProxyURL // 从 session 中获取 redirectTo
	if redirectTo == "" {
		redirectTo = "/dashboard"
	}

	return &HandleCallbackResult{
		Token:      token,
		RedirectTo: redirectTo,
		IsNewUser:  isNewUser,
	}, nil
}

// GetUserBindings 获取用户的第三方账号绑定列表
func (s *SocialOAuthService) GetUserBindings(ctx context.Context, userID int64) ([]UserOAuthBindingInfo, error) {
	bindings, err := s.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]UserOAuthBindingInfo, 0, len(bindings))
	for _, b := range bindings {
		result = append(result, b.ToBindingInfo())
	}
	return result, nil
}

// StartBindResult 启动绑定流程结果
type StartBindResult struct {
	AuthURL   string `json:"auth_url"`
	SessionID string `json:"session_id"`
}

// StartBind 启动账号绑定流程
func (s *SocialOAuthService) StartBind(ctx context.Context, userID int64, providerName, callbackURL string) (*StartBindResult, error) {
	// 检查是否已绑定该提供商
	existing, err := s.bindingRepo.GetByUserIDAndProvider(ctx, userID, providerName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrOAuthBindingExists
	}

	// 获取提供商配置
	provider, err := s.providerRepo.GetByName(ctx, providerName)
	if err != nil {
		return nil, err
	}
	if !provider.Enabled {
		return nil, ErrOAuthProviderDisabled
	}

	// 获取策略
	strategy, ok := s.GetStrategy(providerName)
	if !ok {
		return nil, ErrOAuthProviderNotFound
	}

	// 生成 state 和 session
	state, err := oauth.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	sessionID, err := oauth.GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}

	// 存储 session（使用 CodeVerifier 字段存储 userID）
	session := &oauth.OAuthSession{
		State:        state,
		CodeVerifier: fmt.Sprintf("%d", userID), // 存储 userID
		Scope:        provider.GetConfigString("scopes"),
		CreatedAt:    time.Now(),
	}
	s.sessionStore.Set(sessionID, session)

	// 构建授权 URL
	authURL := strategy.BuildAuthURL(state, callbackURL, provider)

	return &StartBindResult{
		AuthURL:   authURL,
		SessionID: sessionID,
	}, nil
}

// HandleBindCallback 处理绑定回调
func (s *SocialOAuthService) HandleBindCallback(ctx context.Context, providerName, code, state, sessionID, callbackURL string) error {
	// 验证 session
	session, ok := s.sessionStore.Get(sessionID)
	if !ok {
		return ErrOAuthStateMismatch
	}
	defer s.sessionStore.Delete(sessionID)

	if session.State != state {
		return ErrOAuthStateMismatch
	}

	// 从 session 中获取 userID
	var userID int64
	if _, err := fmt.Sscanf(session.CodeVerifier, "%d", &userID); err != nil {
		return ErrOAuthStateMismatch
	}

	// 获取提供商配置
	provider, err := s.providerRepo.GetByName(ctx, providerName)
	if err != nil {
		return err
	}

	// 获取策略并换取用户信息
	strategy, ok := s.GetStrategy(providerName)
	if !ok {
		return ErrOAuthProviderNotFound
	}

	userInfo, err := strategy.ExchangeCode(ctx, code, callbackURL, provider)
	if err != nil {
		log.Printf("[SocialOAuth] bind exchange code failed for %s: %v", providerName, err)
		return ErrOAuthCodeExchangeFailed
	}

	// 检查该第三方账号是否已被其他用户绑定
	existingBinding, err := s.bindingRepo.GetByProviderUserID(ctx, providerName, userInfo.ProviderUserID)
	if err != nil {
		return err
	}
	if existingBinding != nil {
		return ErrOAuthAccountBound
	}

	// 创建绑定
	binding := &UserOAuthBinding{
		UserID:           userID,
		Provider:         providerName,
		ProviderUserID:   userInfo.ProviderUserID,
		ProviderEmail:    stringPtr(userInfo.Email),
		ProviderUsername: stringPtr(userInfo.Username),
		ProviderAvatar:   stringPtr(userInfo.Avatar),
		AccessToken:      stringPtr(userInfo.AccessToken),
		RefreshToken:     stringPtr(userInfo.RefreshToken),
	}

	return s.bindingRepo.Create(ctx, binding)
}

// Unbind 解绑第三方账号
func (s *SocialOAuthService) Unbind(ctx context.Context, userID int64, providerName string) error {
	// 检查绑定是否存在
	binding, err := s.bindingRepo.GetByUserIDAndProvider(ctx, userID, providerName)
	if err != nil {
		return err
	}
	if binding == nil {
		return ErrOAuthBindingNotFound
	}

	// 检查用户是否有密码（如果没有密码且只有一个绑定，不允许解绑）
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 获取绑定数量
	bindingCount, err := s.bindingRepo.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// 如果用户是 OAuth 创建的（无真实密码）且只有一个绑定，不允许解绑
	// 通过检查密码哈希是否以特定前缀开头来判断是否是 OAuth 用户
	// 注意：这里简化处理，实际应该有更好的标识方式
	if bindingCount <= 1 {
		// 检查用户是否有其他登录方式（例如密码登录）
		// 如果邮箱是合成邮箱（@oauth.local），则认为没有密码登录
		if isOAuthSyntheticEmail(user.Email) {
			return ErrOAuthCannotUnbindLast
		}
	}

	return s.bindingRepo.Delete(ctx, userID, providerName)
}

// GetAllProviders 获取所有提供商（管理 API）
func (s *SocialOAuthService) GetAllProviders(ctx context.Context) ([]OAuthProvider, error) {
	return s.providerRepo.List(ctx)
}

// UpdateProvider 更新提供商配置（管理 API）
func (s *SocialOAuthService) UpdateProvider(ctx context.Context, name string, input *UpdateOAuthProviderInput) (*OAuthProvider, error) {
	provider, err := s.providerRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	if input.ClientID != nil {
		provider.ClientID = *input.ClientID
	}
	if input.ClientSecret != nil {
		provider.ClientSecret = *input.ClientSecret
	}
	if input.Enabled != nil {
		provider.Enabled = *input.Enabled
	}
	if input.Config != nil {
		provider.Config = input.Config
	}

	if err := s.providerRepo.Update(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

// Stop 停止服务
func (s *SocialOAuthService) Stop() {
	s.sessionStore.Stop()
}

// Helper functions

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func isOAuthSyntheticEmail(email string) bool {
	return len(email) > 12 && email[len(email)-12:] == "@oauth.local"
}
