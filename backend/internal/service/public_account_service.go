package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PublicAccountService 处理公开的账户贡献功能
type PublicAccountService struct {
	accountRepo             AccountRepository
	groupRepo               GroupRepository
	antigravityOAuthService *AntigravityOAuthService
}

// NewPublicAccountService 创建公开账户服务实例
func NewPublicAccountService(
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	antigravityOAuthService *AntigravityOAuthService,
) *PublicAccountService {
	return &PublicAccountService{
		accountRepo:             accountRepo,
		groupRepo:               groupRepo,
		antigravityOAuthService: antigravityOAuthService,
	}
}

// PublicOAuthStartResult 公开 OAuth 启动结果
type PublicOAuthStartResult struct {
	AuthURL   string `json:"auth_url"`
	SessionID string `json:"session_id"`
	State     string `json:"state"`
}

// GenerateAuthURL 生成公开的 OAuth 授权链接
func (s *PublicAccountService) GenerateAuthURL(ctx context.Context) (*PublicOAuthStartResult, error) {
	result, err := s.antigravityOAuthService.GenerateAuthURL(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("生成授权链接失败: %w", err)
	}

	return &PublicOAuthStartResult{
		AuthURL:   result.AuthURL,
		SessionID: result.SessionID,
		State:     result.State,
	}, nil
}

// PublicOAuthCompleteInput 公开 OAuth 完成输入
type PublicOAuthCompleteInput struct {
	SessionID string
	State     string
	Code      string
}

// PublicOAuthCompleteResult 公开 OAuth 完成结果
type PublicOAuthCompleteResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Email   string `json:"email,omitempty"`
	IsNew   bool   `json:"is_new"` // true=新创建, false=更新已有
}

// CompleteOAuth 完成 OAuth 流程，创建或更新账户
func (s *PublicAccountService) CompleteOAuth(ctx context.Context, input *PublicOAuthCompleteInput) (*PublicOAuthCompleteResult, error) {
	// 1. 交换 token
	tokenInfo, err := s.antigravityOAuthService.ExchangeCode(ctx, &AntigravityExchangeCodeInput{
		SessionID: input.SessionID,
		State:     input.State,
		Code:      input.Code,
		ProxyID:   nil,
	})
	if err != nil {
		return nil, fmt.Errorf("Token 交换失败: %w", err)
	}

	email := strings.TrimSpace(tokenInfo.Email)
	if email == "" {
		return nil, fmt.Errorf("无法获取账户邮箱")
	}

	// 2. 查找默认分组 claude_share
	groupID, err := s.findGroupIDByName(ctx, "claude_share")
	if err != nil {
		return nil, fmt.Errorf("查找默认分组失败: %w", err)
	}

	// 3. 构建凭证
	credentials := s.antigravityOAuthService.BuildAccountCredentials(tokenInfo)

	// 4. 查找是否存在同 email 的账户
	existingAccount, err := s.findAccountByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("查询账户失败: %w", err)
	}

	if existingAccount != nil {
		// 更新已有账户
		existingAccount.Credentials = credentials
		existingAccount.Status = StatusDisabled // 待审核状态
		existingAccount.Schedulable = false
		existingAccount.UpdatedAt = time.Now()

		if err := s.accountRepo.Update(ctx, existingAccount); err != nil {
			return nil, fmt.Errorf("更新账户失败: %w", err)
		}

		return &PublicOAuthCompleteResult{
			Success: true,
			Message: "账户已更新，等待管理员审核",
			Email:   email,
			IsNew:   false,
		}, nil
	}

	// 5. 创建新账户
	now := time.Now()
	account := &Account{
		Name:        email, // 使用邮箱作为名称
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Credentials: credentials,
		Extra:       make(map[string]any),
		Concurrency: 2,              // 默认并发数
		Priority:    50,             // 默认优先级
		Status:      StatusDisabled, // 待审核状态
		Schedulable: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("创建账户失败: %w", err)
	}

	// 6. 绑定分组
	if groupID > 0 {
		if err := s.accountRepo.BindGroups(ctx, account.ID, []int64{groupID}); err != nil {
			// 绑定失败不影响账户创建，仅记录日志
			fmt.Printf("[PublicAccountService] 绑定分组失败: account=%d, group=%d, err=%v\n", account.ID, groupID, err)
		}
	}

	return &PublicOAuthCompleteResult{
		Success: true,
		Message: "账户创建成功，等待管理员审核",
		Email:   email,
		IsNew:   true,
	}, nil
}

// findGroupIDByName 通过名称查找分组 ID
func (s *PublicAccountService) findGroupIDByName(ctx context.Context, name string) (int64, error) {
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return 0, err
	}

	for _, g := range groups {
		if g.Name == name {
			return g.ID, nil
		}
	}

	return 0, nil // 未找到返回 0，不报错
}

// findAccountByEmail 通过 email 查找 Antigravity 账户
func (s *PublicAccountService) findAccountByEmail(ctx context.Context, email string) (*Account, error) {
	accounts, err := s.accountRepo.ListByPlatform(ctx, PlatformAntigravity)
	if err != nil {
		return nil, err
	}

	for i := range accounts {
		acc := &accounts[i]
		accEmail := acc.GetCredential("email")
		if strings.EqualFold(accEmail, email) {
			return acc, nil
		}
	}

	return nil, nil
}
