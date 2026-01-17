package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// GitHub OAuth 端点
const (
	githubAuthURL     = "https://github.com/login/oauth/authorize"
	githubTokenURL    = "https://github.com/login/oauth/access_token"
	githubUserURL     = "https://api.github.com/user"
	githubEmailsURL   = "https://api.github.com/user/emails"
	githubDefaultScopes = "read:user user:email"
)

// HTTP 超时设置
const httpTimeout = 30 * time.Second

// GitHubOAuthStrategy GitHub OAuth 策略实现
type GitHubOAuthStrategy struct{}

// NewGitHubOAuthStrategy 创建 GitHub OAuth 策略
func NewGitHubOAuthStrategy() *GitHubOAuthStrategy {
	return &GitHubOAuthStrategy{}
}

// Name 返回提供商名称
func (s *GitHubOAuthStrategy) Name() string {
	return "github"
}

// BuildAuthURL 构建 GitHub 授权 URL
func (s *GitHubOAuthStrategy) BuildAuthURL(state, redirectURI string, provider *OAuthProvider) string {
	scopes := provider.GetConfigString("scopes")
	if scopes == "" {
		scopes = githubDefaultScopes
	}

	params := url.Values{}
	params.Set("client_id", provider.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", scopes)
	params.Set("state", state)
	params.Set("allow_signup", "true")

	return fmt.Sprintf("%s?%s", githubAuthURL, params.Encode())
}

// ExchangeCode 用授权码换取用户信息
func (s *GitHubOAuthStrategy) ExchangeCode(ctx context.Context, code, redirectURI string, provider *OAuthProvider) (*OAuthUserInfo, error) {
	// 换取 access_token
	accessToken, err := s.exchangeToken(ctx, code, redirectURI, provider)
	if err != nil {
		return nil, fmt.Errorf("exchange token: %w", err)
	}

	// 获取用户信息
	userInfo, err := s.getUserInfo(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}

	// 如果没有公开邮箱，尝试获取主邮箱
	if userInfo.Email == "" {
		email, err := s.getPrimaryEmail(ctx, accessToken)
		if err == nil {
			userInfo.Email = email
		}
	}

	userInfo.AccessToken = accessToken

	return userInfo, nil
}

// githubTokenResponse GitHub token 响应
type githubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// exchangeToken 用授权码换取 token
func (s *GitHubOAuthStrategy) exchangeToken(ctx context.Context, code, redirectURI string, provider *OAuthProvider) (string, error) {
	data := url.Values{}
	data.Set("client_id", provider.ClientID)
	data.Set("client_secret", provider.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: httpTimeout}

	// GitHub 需要 Accept 头来返回 JSON
	formReq, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL, nil)
	if err != nil {
		return "", err
	}
	formReq.Header.Set("Accept", "application/json")
	formReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	formReq.URL.RawQuery = data.Encode()

	// 使用 PostForm
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL+"?"+data.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	var tokenResp githubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	return tokenResp.AccessToken, nil
}

// githubUserResponse GitHub 用户信息响应
type githubUserResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// getUserInfo 获取 GitHub 用户信息
func (s *GitHubOAuthStrategy) getUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user request failed with status: %d", resp.StatusCode)
	}

	var userResp githubUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("decode user response: %w", err)
	}

	// 使用 login 作为用户名，如果有 name 则优先使用
	username := userResp.Login
	if userResp.Name != "" {
		username = userResp.Name
	}

	return &OAuthUserInfo{
		ProviderUserID: strconv.FormatInt(userResp.ID, 10),
		Email:          userResp.Email,
		Username:       username,
		Avatar:         userResp.AvatarURL,
	}, nil
}

// githubEmailResponse GitHub 邮箱响应
type githubEmailResponse struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// getPrimaryEmail 获取用户的主邮箱
func (s *GitHubOAuthStrategy) getPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubEmailsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("emails request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("emails request failed with status: %d", resp.StatusCode)
	}

	var emails []githubEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("decode emails response: %w", err)
	}

	// 找到主邮箱且已验证
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	// 如果没有主邮箱，返回第一个已验证的邮箱
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no verified email found")
}
