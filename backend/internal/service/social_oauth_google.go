package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Google OAuth 端点
const (
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	googleDefaultScopes = "openid email profile"
)

// GoogleOAuthStrategy Google OAuth 策略实现
type GoogleOAuthStrategy struct{}

// NewGoogleOAuthStrategy 创建 Google OAuth 策略
func NewGoogleOAuthStrategy() *GoogleOAuthStrategy {
	return &GoogleOAuthStrategy{}
}

// Name 返回提供商名称
func (s *GoogleOAuthStrategy) Name() string {
	return "google"
}

// BuildAuthURL 构建 Google 授权 URL
func (s *GoogleOAuthStrategy) BuildAuthURL(state, redirectURI string, provider *OAuthProvider) string {
	scopes := provider.GetConfigString("scopes")
	if scopes == "" {
		scopes = googleDefaultScopes
	}

	params := url.Values{}
	params.Set("client_id", provider.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", scopes)
	params.Set("state", state)
	params.Set("access_type", "offline") // 获取 refresh_token
	params.Set("prompt", "consent")      // 强制显示同意页面以获取 refresh_token

	return fmt.Sprintf("%s?%s", googleAuthURL, params.Encode())
}

// ExchangeCode 用授权码换取用户信息
func (s *GoogleOAuthStrategy) ExchangeCode(ctx context.Context, code, redirectURI string, provider *OAuthProvider) (*OAuthUserInfo, error) {
	// 换取 access_token
	tokenResp, err := s.exchangeToken(ctx, code, redirectURI, provider)
	if err != nil {
		return nil, fmt.Errorf("exchange token: %w", err)
	}

	// 获取用户信息
	userInfo, err := s.getUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}

	userInfo.AccessToken = tokenResp.AccessToken
	userInfo.RefreshToken = tokenResp.RefreshToken

	return userInfo, nil
}

// googleTokenResponse Google token 响应
type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token,omitempty"`
}

// exchangeToken 用授权码换取 token
func (s *GoogleOAuthStrategy) exchangeToken(ctx context.Context, code, redirectURI string, provider *OAuthProvider) (*googleTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", provider.ClientID)
	data.Set("client_secret", provider.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = data.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 使用 POST body 而不是 query string
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = http.NoBody

	// 重新构建请求
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.PostForm(googleTokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	var tokenResp googleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	return &tokenResp, nil
}

// googleUserInfoResponse Google 用户信息响应
type googleUserInfoResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// getUserInfo 获取 Google 用户信息
func (s *GoogleOAuthStrategy) getUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed with status: %d", resp.StatusCode)
	}

	var userResp googleUserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("decode userinfo response: %w", err)
	}

	// 构建用户名：优先使用 name，否则使用 email 前缀
	username := userResp.Name
	if username == "" && userResp.Email != "" {
		if idx := len(userResp.Email); idx > 0 {
			for i, c := range userResp.Email {
				if c == '@' {
					username = userResp.Email[:i]
					break
				}
			}
		}
	}

	return &OAuthUserInfo{
		ProviderUserID: userResp.ID,
		Email:          userResp.Email,
		Username:       username,
		Avatar:         userResp.Picture,
	}, nil
}
