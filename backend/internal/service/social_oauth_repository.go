package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrOAuthProviderNotFound   = infraerrors.NotFound("OAUTH_PROVIDER_NOT_FOUND", "oauth provider not found")
	ErrOAuthProviderDisabled   = infraerrors.BadRequest("OAUTH_PROVIDER_DISABLED", "oauth provider is disabled")
	ErrOAuthBindingExists      = infraerrors.Conflict("OAUTH_BINDING_EXISTS", "oauth binding already exists")
	ErrOAuthBindingNotFound    = infraerrors.NotFound("OAUTH_BINDING_NOT_FOUND", "oauth binding not found")
	ErrOAuthCannotUnbindLast   = infraerrors.BadRequest("OAUTH_CANNOT_UNBIND_LAST", "cannot unbind the last login method")
	ErrOAuthAccountBound       = infraerrors.Conflict("OAUTH_ACCOUNT_BOUND", "this account is already bound to another user")
	ErrOAuthStateMismatch      = infraerrors.BadRequest("OAUTH_STATE_MISMATCH", "oauth state mismatch")
	ErrOAuthCodeExchangeFailed = infraerrors.BadRequest("OAUTH_CODE_EXCHANGE_FAILED", "failed to exchange oauth code")
	ErrOAuthUserInfoFailed     = infraerrors.BadRequest("OAUTH_USERINFO_FAILED", "failed to get user info from oauth provider")
)

// OAuthProviderRepository OAuth 提供商仓储接口
type OAuthProviderRepository interface {
	// GetByName 根据名称获取提供商
	GetByName(ctx context.Context, name string) (*OAuthProvider, error)

	// List 获取所有提供商
	List(ctx context.Context) ([]OAuthProvider, error)

	// ListEnabled 获取所有已启用的提供商
	ListEnabled(ctx context.Context) ([]OAuthProvider, error)

	// Update 更新提供商配置
	Update(ctx context.Context, provider *OAuthProvider) error
}

// UserOAuthBindingRepository 用户 OAuth 绑定仓储接口
type UserOAuthBindingRepository interface {
	// GetByProviderUserID 根据提供商和第三方用户ID查找绑定
	GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*UserOAuthBinding, error)

	// GetByUserIDAndProvider 根据用户ID和提供商查找绑定
	GetByUserIDAndProvider(ctx context.Context, userID int64, provider string) (*UserOAuthBinding, error)

	// GetByUserID 获取用户所有绑定
	GetByUserID(ctx context.Context, userID int64) ([]UserOAuthBinding, error)

	// Create 创建绑定
	Create(ctx context.Context, binding *UserOAuthBinding) error

	// Delete 删除绑定
	Delete(ctx context.Context, userID int64, provider string) error

	// CountByUserID 统计用户绑定数量
	CountByUserID(ctx context.Context, userID int64) (int, error)

	// TransferBinding 将绑定转移到新用户（用于强制转移场景）
	TransferBinding(ctx context.Context, provider, providerUserID string, newUserID int64, binding *UserOAuthBinding) error
}
