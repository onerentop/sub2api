package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
)

// PublicAccountService 处理公开的账户贡献功能
type PublicAccountService struct {
	accountRepo             AccountRepository
	groupRepo               GroupRepository
	antigravityOAuthService *AntigravityOAuthService
	httpUpstream            HTTPUpstream
}

// NewPublicAccountService 创建公开账户服务实例
func NewPublicAccountService(
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	antigravityOAuthService *AntigravityOAuthService,
	httpUpstream HTTPUpstream,
) *PublicAccountService {
	return &PublicAccountService{
		accountRepo:             accountRepo,
		groupRepo:               groupRepo,
		antigravityOAuthService: antigravityOAuthService,
		httpUpstream:            httpUpstream,
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
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Email     string `json:"email,omitempty"`
	IsNew     bool   `json:"is_new"`               // true=新创建, false=更新已有
	SessionID string `json:"session_id,omitempty"` // 用于后续唤醒请求（24小时有效）
}

// CompleteOAuth 完成 OAuth 流程，创建或更新账户
// 同时保留 session 用于后续唤醒请求（24 小时有效）
func (s *PublicAccountService) CompleteOAuth(ctx context.Context, input *PublicOAuthCompleteInput) (*PublicOAuthCompleteResult, error) {
	// 1. 交换 token（使用保留 session 的版本）
	tokenInfo, err := s.antigravityOAuthService.ExchangeCodeAndKeepSession(ctx, &AntigravityExchangeCodeInput{
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
		existingAccount.Status = StatusActive // 状态正常，通过 schedulable 控制
		existingAccount.Schedulable = false   // 重新提交后需要管理员再次启用调度
		existingAccount.UpdatedAt = time.Now()

		if err := s.accountRepo.Update(ctx, existingAccount); err != nil {
			return nil, fmt.Errorf("更新账户失败: %w", err)
		}

		return &PublicOAuthCompleteResult{
			Success:   true,
			Message:   "账户已更新，等待管理员审核",
			Email:     email,
			IsNew:     false,
			SessionID: input.SessionID,
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
		Concurrency: 2,             // 默认并发数
		Priority:    50,            // 默认优先级
		Status:      StatusActive,  // 状态正常，通过 schedulable 控制是否启用调度
		Schedulable: false,         // 默认不参与调度，管理员审核后手动启用
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
		Success:   true,
		Message:   "账户创建成功，等待管理员审核",
		Email:     email,
		IsNew:     true,
		SessionID: input.SessionID,
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

// PublicWakeInput 唤醒请求输入
type PublicWakeInput struct {
	SessionID string   `json:"session_id"`
	Models    []string `json:"models,omitempty"` // 可选，默认使用 gemini-3-flash
}

// PublicWakeResult 唤醒请求结果
type PublicWakeResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Model    string `json:"model"`
	Text     string `json:"text,omitempty"`     // AI 响应文本
	Duration int64  `json:"duration,omitempty"` // 耗时 (ms)
}

// Wake 执行唤醒请求（发送测试消息触发配额）
func (s *PublicAccountService) Wake(ctx context.Context, input *PublicWakeInput) (*PublicWakeResult, error) {
	startTime := time.Now()

	// 1. 获取 session
	session, err := s.antigravityOAuthService.GetSessionForWake(input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("session 无效: %w", err)
	}

	// 2. 确定要唤醒的模型
	models := input.Models
	if len(models) == 0 {
		models = []string{"gemini-3-flash"}
	}
	modelID := models[0] // 目前只支持单个模型

	// 3. 检查 token 是否过期，过期则尝试刷新
	accessToken := session.AccessToken
	if session.ExpiresAt > 0 && time.Now().Unix() >= session.ExpiresAt {
		// Token 已过期，尝试刷新
		if strings.TrimSpace(session.RefreshToken) == "" {
			return nil, fmt.Errorf("token 已过期且无 refresh_token")
		}
		tokenInfo, err := s.antigravityOAuthService.RefreshToken(ctx, session.RefreshToken, session.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("token 已过期且刷新失败: %w", err)
		}
		accessToken = tokenInfo.AccessToken

		// 更新 session 中的 token 信息（避免重复刷新，处理 rotating refresh_token）
		s.antigravityOAuthService.UpdateSessionToken(input.SessionID, tokenInfo)
	}

	// 4. 构建唤醒请求
	requestBody, err := s.buildWakeRequest(session.ProjectID, modelID)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}

	// 5. URL fallback 循环
	availableURLs := antigravity.DefaultURLAvailability.GetAvailableURLs()
	if len(availableURLs) == 0 {
		availableURLs = antigravity.BaseURLs
	}
	if len(availableURLs) == 0 {
		return nil, fmt.Errorf("没有可用的 API 端点")
	}

	// 安全截取 session ID 用于日志
	sessionIDShort := input.SessionID
	if len(sessionIDShort) > 8 {
		sessionIDShort = sessionIDShort[:8]
	}

	var lastErr error
	for urlIdx, baseURL := range availableURLs {
		// 构建 HTTP 请求
		req, err := antigravity.NewAPIRequestWithURL(ctx, baseURL, "streamGenerateContent", accessToken, requestBody)
		if err != nil {
			lastErr = err
			continue
		}

		log.Printf("[PublicWake] session=%s model=%s url=%s", sessionIDShort, modelID, req.URL.String())

		// 发送请求（不使用代理，使用 0 作为 accountID 和 concurrency）
		resp, err := s.httpUpstream.Do(req, session.ProxyURL, 0, 1)
		if err != nil {
			lastErr = fmt.Errorf("请求失败: %w", err)
			if shouldAntigravityFallbackToNextURL(err, 0) && urlIdx < len(availableURLs)-1 {
				antigravity.DefaultURLAvailability.MarkUnavailable(baseURL)
				continue
			}
			return nil, lastErr
		}

		// 读取响应
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("读取响应失败: %w", err)
		}

		// 检查 URL 降级
		if shouldAntigravityFallbackToNextURL(nil, resp.StatusCode) && urlIdx < len(availableURLs)-1 {
			antigravity.DefaultURLAvailability.MarkUnavailable(baseURL)
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("API 返回 %d: %s", resp.StatusCode, string(respBody))
		}

		// 解析响应
		text := extractTextFromSSEResponse(respBody)
		duration := time.Since(startTime).Milliseconds()

		return &PublicWakeResult{
			Success:  true,
			Message:  "唤醒成功",
			Model:    modelID,
			Text:     text,
			Duration: duration,
		}, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("所有 API 端点请求失败")
	}
	return nil, lastErr
}

// buildWakeRequest 构建唤醒请求体
func (s *PublicAccountService) buildWakeRequest(projectID, modelID string) ([]byte, error) {
	// 生成请求 ID 和会话 ID
	requestID := fmt.Sprintf("req_%d_%s", time.Now().UnixMilli(), generateShortID())
	sessionID := fmt.Sprintf("sess_%d_%s", time.Now().UnixMilli(), generateShortID())

	payload := map[string]any{
		"project":     projectID,
		"requestId":   requestID,
		"model":       modelID,
		"userAgent":   "antigravity",
		"requestType": "agent",
		"request": map[string]any{
			"contents": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]any{
						{"text": "hi"},
					},
				},
			},
			"session_id": sessionID,
			"systemInstruction": map[string]any{
				"parts": []map[string]any{
					{"text": antigravity.GetDefaultIdentityPatch()},
				},
			},
			"generationConfig": map[string]any{
				"temperature": 0,
			},
		},
	}
	return json.Marshal(payload)
}

// generateShortID 生成 6 位随机字符串
func generateShortID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	randBytes := make([]byte, 6)
	_, _ = rand.Read(randBytes)
	for i := range b {
		b[i] = charset[int(randBytes[i])%len(charset)]
	}
	return string(b)
}
