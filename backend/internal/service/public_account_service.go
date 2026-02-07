package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
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
		Concurrency: 2,            // 默认并发数
		Priority:    50,           // 默认优先级
		Status:      StatusActive, // 状态正常，通过 schedulable 控制是否启用调度
		Schedulable: false,        // 默认不参与调度，管理员审核后手动启用
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
	SessionID       string   `json:"session_id"`
	Models          []string `json:"models,omitempty"`            // 可选，默认使用 gemini-3-flash
	CustomPrompt    string   `json:"custom_prompt,omitempty"`     // 自定义唤醒词，默认 "hi"
	MaxOutputTokens int      `json:"max_output_tokens,omitempty"` // 最大输出 token 数，0 表示不限制
}

// wakeSystemPrompt 唤醒请求的系统提示词
// 与 vscode-antigravity-cockpit/src/auto_trigger/trigger_service.ts ANTIGRAVITY_SYSTEM_PROMPT 保持一致
const wakeSystemPrompt = "You are Antigravity, a powerful agentic AI coding assistant designed by the Google Deepmind team working on Advanced Agentic Coding.You are pair programming with a USER to solve their coding task. The task may require creating a new codebase, modifying or debugging an existing codebase, or simply answering a question.**Absolute paths only****Proactiveness**"

// WakeModelResult 单个模型唤醒结果
type WakeModelResult struct {
	Model            string `json:"model"`
	Success          bool   `json:"success"`
	Message          string `json:"message,omitempty"`
	Text             string `json:"text,omitempty"`
	Duration         int64  `json:"duration,omitempty"`          // 耗时 (ms)
	PromptTokens     int    `json:"prompt_tokens,omitempty"`     // 输入 token 数
	CompletionTokens int    `json:"completion_tokens,omitempty"` // 输出 token 数
	TotalTokens      int    `json:"total_tokens,omitempty"`      // 总 token 数
	TraceID          string `json:"trace_id,omitempty"`          // 调试用 traceId
}

// PublicWakeResult 唤醒请求结果
type PublicWakeResult struct {
	Success  bool               `json:"success"`
	Message  string             `json:"message,omitempty"`
	Duration int64              `json:"duration,omitempty"` // 总耗时 (ms)
	Results  []*WakeModelResult `json:"results,omitempty"`  // 多模型结果（新增）
	// 兼容字段：保留第一个成功结果的信息
	Model string `json:"model"`
	Text  string `json:"text,omitempty"`
}

// Wake 执行唤醒请求（发送测试消息触发配额）
// 支持多模型并发执行，参考 vscode-antigravity-cockpit/src/auto_trigger/trigger_service.ts
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

	// 3. 检查 token 是否过期，过期则尝试刷新
	accessToken := session.AccessToken
	if session.ExpiresAt > 0 && time.Now().Unix() >= session.ExpiresAt {
		if strings.TrimSpace(session.RefreshToken) == "" {
			return nil, fmt.Errorf("token 已过期且无 refresh_token")
		}
		tokenInfo, err := s.antigravityOAuthService.RefreshToken(ctx, session.RefreshToken, session.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("token 已过期且刷新失败: %w", err)
		}
		accessToken = tokenInfo.AccessToken
		s.antigravityOAuthService.UpdateSessionToken(input.SessionID, tokenInfo)
	}

	// 安全截取 session ID 用于日志
	sessionIDShort := input.SessionID
	if len(sessionIDShort) > 8 {
		sessionIDShort = sessionIDShort[:8]
	}

	// 4. 【关键】调用 ResolveProjectId 确保账户激活
	// 与 vscode-antigravity-cockpit 保持一致：每次 trigger 前都调用 loadCodeAssist + onboardUser
	// 这是触发配额的必要步骤！
	client := antigravity.NewClient(session.ProxyURL)
	projectId, tier, err := client.ResolveProjectId(ctx, accessToken)
	if err != nil {
		log.Printf("[PublicWake] session=%s ResolveProjectId 失败: %v (使用缓存的 projectId)", sessionIDShort, err)
		// 降级使用 session 中的 projectId
		projectId = session.ProjectID
	} else {
		log.Printf("[PublicWake] session=%s ResolveProjectId 成功: projectId=%s, tier=%s", sessionIDShort, projectId, tier)
		// 更新 session 中的 projectId（如果有变化）
		if projectId != "" && projectId != session.ProjectID {
			session.ProjectID = projectId
		}
	}

	// 5. 并发执行多模型唤醒（最多 4 个并发）
	const maxConcurrency = 4
	results := make([]*WakeModelResult, len(models))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)

	for i, modelID := range models {
		wg.Add(1)
		go func(idx int, model string) {
			defer wg.Done()
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			result := s.wakeModel(ctx, session, accessToken, model, input.CustomPrompt, input.MaxOutputTokens, sessionIDShort)
			results[idx] = result
		}(i, modelID)
	}
	wg.Wait()

	// 6. 聚合结果
	return s.aggregateWakeResults(results, time.Since(startTime).Milliseconds()), nil
}

// wakeModel 执行单个模型的唤醒请求
func (s *PublicAccountService) wakeModel(
	ctx context.Context,
	session *antigravity.OAuthSession,
	accessToken, modelID, customPrompt string,
	maxOutputTokens int,
	sessionIDShort string,
) *WakeModelResult {
	modelStart := time.Now()

	// 构建唤醒请求
	requestBody, err := s.buildWakeRequest(session.ProjectID, modelID, customPrompt, maxOutputTokens)
	if err != nil {
		return &WakeModelResult{
			Model:    modelID,
			Success:  false,
			Message:  fmt.Sprintf("构建请求失败: %v", err),
			Duration: time.Since(modelStart).Milliseconds(),
		}
	}

	// URL fallback 循环
	availableURLs := antigravity.DefaultURLAvailability.GetAvailableURLs()
	if len(availableURLs) == 0 {
		availableURLs = antigravity.BaseURLs
	}
	if len(availableURLs) == 0 {
		return &WakeModelResult{
			Model:    modelID,
			Success:  false,
			Message:  "没有可用的 API 端点",
			Duration: time.Since(modelStart).Milliseconds(),
		}
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

		// 发送请求
		resp, err := s.httpUpstream.Do(req, session.ProxyURL, 0, 1)
		if err != nil {
			lastErr = fmt.Errorf("请求失败: %w", err)
			if shouldAntigravityFallbackToNextURL(err, 0) && urlIdx < len(availableURLs)-1 {
				antigravity.DefaultURLAvailability.MarkUnavailable(baseURL)
				continue
			}
			return &WakeModelResult{
				Model:    modelID,
				Success:  false,
				Message:  lastErr.Error(),
				Duration: time.Since(modelStart).Milliseconds(),
			}
		}

		// 读取响应
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		if err != nil {
			return &WakeModelResult{
				Model:    modelID,
				Success:  false,
				Message:  fmt.Sprintf("读取响应失败: %v", err),
				Duration: time.Since(modelStart).Milliseconds(),
			}
		}

		// 检查 URL 降级
		if shouldAntigravityFallbackToNextURL(nil, resp.StatusCode) && urlIdx < len(availableURLs)-1 {
			antigravity.DefaultURLAvailability.MarkUnavailable(baseURL)
			continue
		}

		if resp.StatusCode >= 400 {
			return &WakeModelResult{
				Model:    modelID,
				Success:  false,
				Message:  fmt.Sprintf("API 返回 %d: %s", resp.StatusCode, string(respBody)),
				Duration: time.Since(modelStart).Milliseconds(),
			}
		}

		// 解析响应（使用增强版解析函数）
		parsed := parseWakeSSEResponse(respBody)
		return &WakeModelResult{
			Model:            modelID,
			Success:          true,
			Message:          "唤醒成功",
			Text:             parsed.Text,
			Duration:         time.Since(modelStart).Milliseconds(),
			PromptTokens:     parsed.PromptTokens,
			CompletionTokens: parsed.CompletionTokens,
			TotalTokens:      parsed.TotalTokens,
			TraceID:          parsed.TraceID,
		}
	}

	// 所有 URL 都失败
	errMsg := "所有 API 端点请求失败"
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	return &WakeModelResult{
		Model:    modelID,
		Success:  false,
		Message:  errMsg,
		Duration: time.Since(modelStart).Milliseconds(),
	}
}

// aggregateWakeResults 聚合多模型唤醒结果
func (s *PublicAccountService) aggregateWakeResults(results []*WakeModelResult, totalDuration int64) *PublicWakeResult {
	var (
		anySuccess   bool
		firstSuccess *WakeModelResult
		successMsgs  []string
		failureMsgs  []string
	)

	for _, r := range results {
		if r.Success {
			anySuccess = true
			if firstSuccess == nil {
				firstSuccess = r
			}
			tokenInfo := ""
			if r.TotalTokens > 0 {
				tokenInfo = fmt.Sprintf(", tokens=%d+%d=%d", r.PromptTokens, r.CompletionTokens, r.TotalTokens)
			}
			traceInfo := ""
			if r.TraceID != "" {
				traceInfo = fmt.Sprintf(", traceId=%s", r.TraceID)
			}
			successMsgs = append(successMsgs, fmt.Sprintf("[%s]: %dms%s%s", r.Model, r.Duration, tokenInfo, traceInfo))
		} else {
			failureMsgs = append(failureMsgs, fmt.Sprintf("[%s]: %s", r.Model, r.Message))
		}
	}

	// 构建汇总消息
	var message string
	if anySuccess && len(failureMsgs) == 0 {
		message = fmt.Sprintf("唤醒成功: %s", strings.Join(successMsgs, "; "))
	} else if anySuccess {
		message = fmt.Sprintf("部分成功: %s; 失败: %s", strings.Join(successMsgs, "; "), strings.Join(failureMsgs, "; "))
	} else {
		message = fmt.Sprintf("唤醒失败: %s", strings.Join(failureMsgs, "; "))
	}

	// 构建结果
	result := &PublicWakeResult{
		Success:  anySuccess,
		Message:  message,
		Duration: totalDuration,
		Results:  results,
	}

	// 兼容字段：保留第一个成功结果的信息
	if firstSuccess != nil {
		result.Model = firstSuccess.Model
		result.Text = firstSuccess.Text
	} else if len(results) > 0 {
		result.Model = results[0].Model
	}

	return result
}

// buildWakeRequest 构建唤醒请求体
func (s *PublicAccountService) buildWakeRequest(projectID, modelID, customPrompt string, maxOutputTokens int) ([]byte, error) {
	// 生成请求 ID 和会话 ID
	requestID := fmt.Sprintf("req_%d_%s", time.Now().UnixMilli(), generateShortID())
	sessionID := fmt.Sprintf("sess_%d_%s", time.Now().UnixMilli(), generateShortID())

	// 确定唤醒词
	prompt := "hi"
	if strings.TrimSpace(customPrompt) != "" {
		prompt = customPrompt
	}

	// 构建生成配置
	generationConfig := map[string]any{
		"temperature": 0,
	}
	if maxOutputTokens > 0 {
		generationConfig["maxOutputTokens"] = maxOutputTokens
	}

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
						{"text": prompt},
					},
				},
			},
			"session_id": sessionID,
			"systemInstruction": map[string]any{
				"parts": []map[string]any{
					{"text": wakeSystemPrompt},
				},
			},
			"generationConfig": generationConfig,
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

// parsedWakeSSEResult SSE 响应解析结果
type parsedWakeSSEResult struct {
	Text             string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	TraceID          string
}

// parseWakeSSEResponse 从 SSE 流式响应中提取完整信息
// 参考 vscode-antigravity-cockpit/src/auto_trigger/trigger_service.ts:parseStreamResult
func parseWakeSSEResponse(respBody []byte) *parsedWakeSSEResult {
	result := &parsedWakeSSEResult{}
	var textParts []string
	lines := bytes.Split(respBody, []byte("\n"))

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// 跳过 SSE 前缀
		if bytes.HasPrefix(line, []byte("data:")) {
			line = bytes.TrimPrefix(line, []byte("data:"))
			line = bytes.TrimSpace(line)
		}

		// 跳过 [DONE] 标记和非 JSON 行
		if len(line) == 0 || line[0] != '{' {
			continue
		}

		// 解析 JSON
		var data map[string]any
		if err := json.Unmarshal(line, &data); err != nil {
			continue
		}

		// 提取 traceId（顶层）
		if traceID, ok := data["traceId"].(string); ok && result.TraceID == "" {
			result.TraceID = traceID
		}

		// 确定 response 对象位置
		response, ok := data["response"].(map[string]any)
		if !ok {
			response = data // 某些响应格式直接在顶层
		}

		// 提取 usageMetadata
		usageMetadata, _ := response["usageMetadata"].(map[string]any)
		if usageMetadata == nil {
			usageMetadata, _ = data["usageMetadata"].(map[string]any)
		}
		if usageMetadata != nil {
			if v, ok := usageMetadata["promptTokenCount"].(float64); ok && result.PromptTokens == 0 {
				result.PromptTokens = int(v)
			}
			if v, ok := usageMetadata["candidatesTokenCount"].(float64); ok && result.CompletionTokens == 0 {
				result.CompletionTokens = int(v)
			}
			if v, ok := usageMetadata["totalTokenCount"].(float64); ok && result.TotalTokens == 0 {
				result.TotalTokens = int(v)
			}
		}

		// 提取文本内容
		candidates, ok := response["candidates"].([]any)
		if !ok || len(candidates) == 0 {
			continue
		}

		candidate, ok := candidates[0].(map[string]any)
		if !ok {
			continue
		}

		content, ok := candidate["content"].(map[string]any)
		if !ok {
			continue
		}

		parts, ok := content["parts"].([]any)
		if !ok {
			continue
		}

		for _, part := range parts {
			if partMap, ok := part.(map[string]any); ok {
				// 跳过 thinking 内容（Gemini thinking 模型的思考过程）
				if thought, ok := partMap["thought"].(bool); ok && thought {
					continue
				}
				if text, ok := partMap["text"].(string); ok && text != "" {
					textParts = append(textParts, text)
				}
			}
		}
	}

	result.Text = strings.Join(textParts, "")
	if result.Text == "" {
		result.Text = "(无回复)"
	}
	return result
}
