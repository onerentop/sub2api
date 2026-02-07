package service

import (
	"time"
)

// OAuthProvider OAuth 提供商配置
type OAuthProvider struct {
	ID           int64
	Name         string // google, github, qq, wechat
	DisplayName  string // Google, GitHub
	ClientID     string
	ClientSecret string
	Enabled      bool
	Config       map[string]any // 额外配置: scopes, endpoints 等
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// GetConfigString 从配置中获取字符串值
func (p *OAuthProvider) GetConfigString(key string) string {
	if p.Config == nil {
		return ""
	}
	if v, ok := p.Config[key].(string); ok {
		return v
	}
	return ""
}

// UserOAuthBinding 用户 OAuth 绑定
type UserOAuthBinding struct {
	ID               int64
	UserID           int64
	Provider         string  // google, github
	ProviderUserID   string  // 第三方平台用户ID
	ProviderEmail    *string // 第三方平台邮箱
	ProviderUsername *string // 第三方平台用户名
	ProviderAvatar   *string // 第三方平台头像URL
	AccessToken      *string // 访问令牌（可选存储）
	RefreshToken     *string // 刷新令牌（可选存储）
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// OAuthUserInfo 统一的第三方用户信息结构
type OAuthUserInfo struct {
	ProviderUserID string
	Email          string
	Username       string
	Avatar         string
	AccessToken    string
	RefreshToken   string
}

// UpdateOAuthProviderInput 更新提供商配置输入
type UpdateOAuthProviderInput struct {
	ClientID     *string
	ClientSecret *string
	Enabled      *bool
	Config       map[string]any
}

// OAuthProviderPublicInfo 公开的提供商信息（不含敏感数据）
type OAuthProviderPublicInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Enabled     bool   `json:"enabled"`
}

// UserOAuthBindingInfo 用户绑定信息（前端展示用）
type UserOAuthBindingInfo struct {
	Provider         string    `json:"provider"`
	ProviderEmail    string    `json:"provider_email,omitempty"`
	ProviderUsername string    `json:"provider_username,omitempty"`
	ProviderAvatar   string    `json:"provider_avatar,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// ToPublicInfo 转换为公开信息
func (p *OAuthProvider) ToPublicInfo() OAuthProviderPublicInfo {
	return OAuthProviderPublicInfo{
		Name:        p.Name,
		DisplayName: p.DisplayName,
		Enabled:     p.Enabled,
	}
}

// ToBindingInfo 转换为绑定信息
func (b *UserOAuthBinding) ToBindingInfo() UserOAuthBindingInfo {
	info := UserOAuthBindingInfo{
		Provider:  b.Provider,
		CreatedAt: b.CreatedAt,
	}
	if b.ProviderEmail != nil {
		info.ProviderEmail = *b.ProviderEmail
	}
	if b.ProviderUsername != nil {
		info.ProviderUsername = *b.ProviderUsername
	}
	if b.ProviderAvatar != nil {
		info.ProviderAvatar = *b.ProviderAvatar
	}
	return info
}
