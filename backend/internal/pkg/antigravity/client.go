// Package antigravity provides a client for the Antigravity API.
package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// resolveHost 从 URL 解析 host
func resolveHost(urlStr string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return parsed.Host
}

// NewAPIRequestWithURL 使用指定的 base URL 创建 Antigravity API 请求（v1internal 端点）
func NewAPIRequestWithURL(ctx context.Context, baseURL, action, accessToken string, body []byte) (*http.Request, error) {
	// 构建 URL，流式请求添加 ?alt=sse 参数
	apiURL := fmt.Sprintf("%s/v1internal:%s", baseURL, action)
	isStream := action == "streamGenerateContent"
	if isStream {
		apiURL += "?alt=sse"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 基础 Headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", UserAgent)

	// Accept Header 根据请求类型设置
	if isStream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}

	// 显式设置 Host Header
	if host := resolveHost(apiURL); host != "" {
		req.Host = host
	}

	return req, nil
}

// NewAPIRequest 使用默认 URL 创建 Antigravity API 请求（v1internal 端点）
// 向后兼容：仅使用默认 BaseURL
func NewAPIRequest(ctx context.Context, action, accessToken string, body []byte) (*http.Request, error) {
	return NewAPIRequestWithURL(ctx, BaseURL, action, accessToken, body)
}

// TokenResponse Google OAuth token 响应
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// UserInfo Google 用户信息
type UserInfo struct {
	Email      string `json:"email"`
	Name       string `json:"name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
	Picture    string `json:"picture,omitempty"`
}

// LoadCodeAssistRequest loadCodeAssist 请求
// 与 vscode-antigravity-cockpit 保持一致，metadata 包含三个字段
type LoadCodeAssistRequest struct {
	Metadata LoadCodeAssistMetadata `json:"metadata"`
}

// LoadCodeAssistMetadata loadCodeAssist 的 metadata
// 必须与 vscode 插件的 CLOUDCODE_METADATA 完全一致
type LoadCodeAssistMetadata struct {
	IDEType    string `json:"ideType"`
	Platform   string `json:"platform"`
	PluginType string `json:"pluginType"`
}

// TierInfo 账户类型信息
type TierInfo struct {
	ID          string `json:"id"`          // free-tier, g1-pro-tier, g1-ultra-tier
	Name        string `json:"name"`        // 显示名称
	Description string `json:"description"` // 描述
}

// UnmarshalJSON supports both legacy string tiers and object tiers.
func (t *TierInfo) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '"' {
		var id string
		if err := json.Unmarshal(data, &id); err != nil {
			return err
		}
		t.ID = id
		return nil
	}
	type alias TierInfo
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*t = TierInfo(decoded)
	return nil
}

// IneligibleTier 不符合条件的层级信息
type IneligibleTier struct {
	Tier *TierInfo `json:"tier,omitempty"`
	// ReasonCode 不符合条件的原因代码，如 INELIGIBLE_ACCOUNT
	ReasonCode    string `json:"reasonCode,omitempty"`
	ReasonMessage string `json:"reasonMessage,omitempty"`
}

// LoadCodeAssistResponse loadCodeAssist 响应
type LoadCodeAssistResponse struct {
	CloudAICompanionProject string            `json:"cloudaicompanionProject"`
	CurrentTier             *TierInfo         `json:"currentTier,omitempty"`
	PaidTier                *TierInfo         `json:"paidTier,omitempty"`
	IneligibleTiers         []*IneligibleTier `json:"ineligibleTiers,omitempty"`
}

// GetTier 获取账户类型
// 优先返回 paidTier（付费订阅级别），否则返回 currentTier
func (r *LoadCodeAssistResponse) GetTier() string {
	if r.PaidTier != nil && r.PaidTier.ID != "" {
		return r.PaidTier.ID
	}
	if r.CurrentTier != nil {
		return r.CurrentTier.ID
	}
	return ""
}

// Client Antigravity API 客户端
type Client struct {
	httpClient *http.Client
}

func NewClient(proxyURL string) *Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if strings.TrimSpace(proxyURL) != "" {
		if proxyURLParsed, err := url.Parse(proxyURL); err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURLParsed),
			}
		}
	}

	return &Client{
		httpClient: client,
	}
}

// isConnectionError 判断是否为连接错误（网络超时、DNS 失败、连接拒绝）
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// 检查超时错误
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// 检查连接错误（DNS 失败、连接拒绝）
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// 检查 URL 错误
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}

// shouldFallbackToNextURL 判断是否应切换到下一个 URL
// 仅连接错误和 HTTP 429 触发 URL 降级
func shouldFallbackToNextURL(err error, statusCode int) bool {
	if isConnectionError(err) {
		return true
	}
	return statusCode == http.StatusTooManyRequests
}

// ExchangeCode 用 authorization code 交换 token
func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("client_id", ClientID)
	params.Set("client_secret", ClientSecret)
	params.Set("code", code)
	params.Set("redirect_uri", RedirectURI)
	params.Set("grant_type", "authorization_code")
	params.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token 交换请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token 交换失败 (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return nil, fmt.Errorf("token 解析失败: %w", err)
	}

	return &tokenResp, nil
}

// RefreshToken 刷新 access_token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("client_id", ClientID)
	params.Set("client_secret", ClientSecret)
	params.Set("refresh_token", refreshToken)
	params.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token 刷新请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token 刷新失败 (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return nil, fmt.Errorf("token 解析失败: %w", err)
	}

	return &tokenResp, nil
}

// GetUserInfo 获取用户信息
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("用户信息请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取用户信息失败 (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var userInfo UserInfo
	if err := json.Unmarshal(bodyBytes, &userInfo); err != nil {
		return nil, fmt.Errorf("用户信息解析失败: %w", err)
	}

	return &userInfo, nil
}

// LoadCodeAssist 获取账户信息，返回解析后的结构体和原始 JSON
// 支持 URL fallback：prod → sandbox
func (c *Client) LoadCodeAssist(ctx context.Context, accessToken string) (*LoadCodeAssistResponse, map[string]any, error) {
	// metadata 必须与 vscode-antigravity-cockpit 的 CLOUDCODE_METADATA 完全一致
	reqBody := LoadCodeAssistRequest{
		Metadata: LoadCodeAssistMetadata{
			IDEType:    "ANTIGRAVITY",
			Platform:   "PLATFORM_UNSPECIFIED",
			PluginType: "GEMINI",
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 获取可用的 URL 列表
	availableURLs := DefaultURLAvailability.GetAvailableURLs()
	if len(availableURLs) == 0 {
		availableURLs = BaseURLs // 所有 URL 都不可用时，重试所有
	}

	var lastErr error
	for urlIdx, baseURL := range availableURLs {
		apiURL := baseURL + "/v1internal:loadCodeAssist"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(bodyBytes)))
		if err != nil {
			lastErr = fmt.Errorf("创建请求失败: %w", err)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", UserAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("loadCodeAssist 请求失败: %w", err)
			if shouldFallbackToNextURL(err, 0) && urlIdx < len(availableURLs)-1 {
				DefaultURLAvailability.MarkUnavailable(baseURL)
				log.Printf("[antigravity] loadCodeAssist URL fallback: %s -> %s", baseURL, availableURLs[urlIdx+1])
				continue
			}
			return nil, nil, lastErr
		}

		respBodyBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close() // 立即关闭，避免循环内 defer 导致的资源泄漏
		if err != nil {
			return nil, nil, fmt.Errorf("读取响应失败: %w", err)
		}

		// 检查是否需要 URL 降级
		if shouldFallbackToNextURL(nil, resp.StatusCode) && urlIdx < len(availableURLs)-1 {
			DefaultURLAvailability.MarkUnavailable(baseURL)
			log.Printf("[antigravity] loadCodeAssist URL fallback (HTTP %d): %s -> %s", resp.StatusCode, baseURL, availableURLs[urlIdx+1])
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, nil, fmt.Errorf("loadCodeAssist 失败 (HTTP %d): %s", resp.StatusCode, string(respBodyBytes))
		}

		var loadResp LoadCodeAssistResponse
		if err := json.Unmarshal(respBodyBytes, &loadResp); err != nil {
			return nil, nil, fmt.Errorf("响应解析失败: %w", err)
		}

		// 解析原始 JSON 为 map
		var rawResp map[string]any
		_ = json.Unmarshal(respBodyBytes, &rawResp)

		return &loadResp, rawResp, nil
	}

	return nil, nil, lastErr
}

// ModelQuotaInfo 模型配额信息
type ModelQuotaInfo struct {
	RemainingFraction float64 `json:"remainingFraction"`
	ResetTime         string  `json:"resetTime,omitempty"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	QuotaInfo *ModelQuotaInfo `json:"quotaInfo,omitempty"`
}

// FetchAvailableModelsRequest fetchAvailableModels 请求
type FetchAvailableModelsRequest struct {
	Project string `json:"project"`
}

// FetchAvailableModelsResponse fetchAvailableModels 响应
type FetchAvailableModelsResponse struct {
	Models map[string]ModelInfo `json:"models"`
}

// FetchAvailableModels 获取可用模型和配额信息，返回解析后的结构体和原始 JSON
// 支持 URL fallback：sandbox → daily → prod
func (c *Client) FetchAvailableModels(ctx context.Context, accessToken, projectID string) (*FetchAvailableModelsResponse, map[string]any, error) {
	reqBody := FetchAvailableModelsRequest{Project: projectID}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 获取可用的 URL 列表
	availableURLs := DefaultURLAvailability.GetAvailableURLs()
	if len(availableURLs) == 0 {
		availableURLs = BaseURLs // 所有 URL 都不可用时，重试所有
	}

	var lastErr error
	for urlIdx, baseURL := range availableURLs {
		apiURL := baseURL + "/v1internal:fetchAvailableModels"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(bodyBytes)))
		if err != nil {
			lastErr = fmt.Errorf("创建请求失败: %w", err)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", UserAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("fetchAvailableModels 请求失败: %w", err)
			if shouldFallbackToNextURL(err, 0) && urlIdx < len(availableURLs)-1 {
				DefaultURLAvailability.MarkUnavailable(baseURL)
				log.Printf("[antigravity] fetchAvailableModels URL fallback: %s -> %s", baseURL, availableURLs[urlIdx+1])
				continue
			}
			return nil, nil, lastErr
		}

		respBodyBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close() // 立即关闭，避免循环内 defer 导致的资源泄漏
		if err != nil {
			return nil, nil, fmt.Errorf("读取响应失败: %w", err)
		}

		// 检查是否需要 URL 降级
		if shouldFallbackToNextURL(nil, resp.StatusCode) && urlIdx < len(availableURLs)-1 {
			DefaultURLAvailability.MarkUnavailable(baseURL)
			log.Printf("[antigravity] fetchAvailableModels URL fallback (HTTP %d): %s -> %s", resp.StatusCode, baseURL, availableURLs[urlIdx+1])
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, nil, fmt.Errorf("fetchAvailableModels 失败 (HTTP %d): %s", resp.StatusCode, string(respBodyBytes))
		}

		var modelsResp FetchAvailableModelsResponse
		if err := json.Unmarshal(respBodyBytes, &modelsResp); err != nil {
			return nil, nil, fmt.Errorf("响应解析失败: %w", err)
		}

		// 解析原始 JSON 为 map
		var rawResp map[string]any
		_ = json.Unmarshal(respBodyBytes, &rawResp)

		return &modelsResp, rawResp, nil
	}

	return nil, nil, lastErr
}

// ==================== OnboardUser API (账号激活) ====================
// 以下是从 vscode-antigravity-cockpit 移植的账号激活逻辑
// 新 OAuth 账号必须调用 onboardUser 才能正常使用 Antigravity 模型

// OnboardMetadata 激活请求的元数据
type OnboardMetadata struct {
	IDEType    string `json:"ideType"`
	Platform   string `json:"platform"`
	PluginType string `json:"pluginType"`
}

// OnboardUserRequest onboardUser 请求
type OnboardUserRequest struct {
	TierID   string          `json:"tierId"`
	Metadata OnboardMetadata `json:"metadata"`
}

// OnboardUserResponseData onboardUser 响应中的 response 字段
type OnboardUserResponseData struct {
	CloudAICompanionProject string `json:"cloudaicompanionProject"`
}

// OnboardUserResponse onboardUser 响应
type OnboardUserResponse struct {
	Done     bool                     `json:"done"`
	Response *OnboardUserResponseData `json:"response,omitempty"`
}

// onboardUser 激活参数
const (
	OnboardMaxAttempts = 5               // 最大轮询次数
	OnboardDelayMS     = 2 * time.Second // 轮询间隔
)

// OnboardUser 激活新 OAuth 账号
// 新账号首次使用需要调用此 API 进行激活，否则无法使用 Antigravity 模型
// tierId: 账户级别 (free-tier, g1-pro-tier, g1-ultra-tier)
// 返回激活后的 projectId，失败返回空字符串
func (c *Client) OnboardUser(ctx context.Context, accessToken, tierId string) (string, error) {
	reqBody := OnboardUserRequest{
		TierID: tierId,
		Metadata: OnboardMetadata{
			IDEType:    "ANTIGRAVITY",
			Platform:   "PLATFORM_UNSPECIFIED",
			PluginType: "GEMINI",
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 获取可用的 URL 列表
	availableURLs := DefaultURLAvailability.GetAvailableURLs()
	if len(availableURLs) == 0 {
		availableURLs = BaseURLs
	}

	// 轮询尝试激活（最多5次，每次间隔2秒）
	for attempt := 1; attempt <= OnboardMaxAttempts; attempt++ {
		var lastErr error
		var done bool
		var projectId string

		for urlIdx, baseURL := range availableURLs {
			apiURL := baseURL + "/v1internal:onboardUser"
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(bodyBytes)))
			if err != nil {
				lastErr = fmt.Errorf("创建请求失败: %w", err)
				continue
			}
			req.Header.Set("Authorization", "Bearer "+accessToken)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", UserAgent)

			resp, err := c.httpClient.Do(req)
			if err != nil {
				lastErr = fmt.Errorf("onboardUser 请求失败: %w", err)
				if shouldFallbackToNextURL(err, 0) && urlIdx < len(availableURLs)-1 {
					DefaultURLAvailability.MarkUnavailable(baseURL)
					log.Printf("[antigravity] onboardUser URL fallback: %s -> %s", baseURL, availableURLs[urlIdx+1])
					continue
				}
				break
			}

			respBodyBytes, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				lastErr = fmt.Errorf("读取响应失败: %w", err)
				break
			}

			// 检查是否需要 URL 降级
			if shouldFallbackToNextURL(nil, resp.StatusCode) && urlIdx < len(availableURLs)-1 {
				DefaultURLAvailability.MarkUnavailable(baseURL)
				log.Printf("[antigravity] onboardUser URL fallback (HTTP %d): %s -> %s", resp.StatusCode, baseURL, availableURLs[urlIdx+1])
				continue
			}

			if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("onboardUser 失败 (HTTP %d): %s", resp.StatusCode, string(respBodyBytes))
				break
			}

			var onboardResp OnboardUserResponse
			if err := json.Unmarshal(respBodyBytes, &onboardResp); err != nil {
				lastErr = fmt.Errorf("响应解析失败: %w", err)
				break
			}

			done = onboardResp.Done
			if done && onboardResp.Response != nil {
				projectId = extractProjectIdFromPath(onboardResp.Response.CloudAICompanionProject)
			}
			break // 请求成功，跳出 URL 循环
		}

		// 如果激活完成，返回 projectId
		if done {
			log.Printf("[antigravity] onboardUser 激活成功 (attempt %d/%d), projectId: %s", attempt, OnboardMaxAttempts, projectId)
			return projectId, nil
		}

		// 未完成且还有重试机会，等待后继续
		if attempt < OnboardMaxAttempts {
			log.Printf("[antigravity] onboardUser 激活中 (attempt %d/%d), 等待 %v 后重试...", attempt, OnboardMaxAttempts, OnboardDelayMS)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(OnboardDelayMS):
			}
		} else if lastErr != nil {
			return "", lastErr
		}
	}

	return "", fmt.Errorf("onboardUser 激活超时（%d 次尝试均未完成）", OnboardMaxAttempts)
}

// extractProjectIdFromPath 从 cloudaicompanionProject 路径中提取 projectId
// 格式: "projects/{projectId}/locations/global/..." 或直接返回字符串
func extractProjectIdFromPath(path string) string {
	if path == "" {
		return ""
	}
	// 尝试解析 projects/{id}/... 格式
	const prefix = "projects/"
	if idx := strings.Index(path, prefix); idx >= 0 {
		rest := path[idx+len(prefix):]
		if slashIdx := strings.Index(rest, "/"); slashIdx > 0 {
			return rest[:slashIdx]
		}
		return rest
	}
	return path
}

// ResolveProjectId 获取账号的 projectId，如果没有则自动激活
// 这是 vscode-antigravity-cockpit 的核心逻辑：先 loadCodeAssist 获取，没有就 onboardUser 激活
func (c *Client) ResolveProjectId(ctx context.Context, accessToken string) (projectId string, tier string, err error) {
	// 1. 先尝试 loadCodeAssist 获取现有 projectId
	loadResp, _, err := c.LoadCodeAssist(ctx, accessToken)
	if err != nil {
		return "", "", fmt.Errorf("loadCodeAssist 失败: %w", err)
	}

	// 获取 tier 信息
	tier = loadResp.GetTier()
	if tier == "" {
		tier = "free-tier" // 默认 free-tier
	}

	// 提取 projectId
	projectId = extractProjectIdFromPath(loadResp.CloudAICompanionProject)

	// 2. 如果已有 projectId，直接返回
	if projectId != "" {
		log.Printf("[antigravity] ResolveProjectId: 已有 projectId=%s, tier=%s", projectId, tier)
		return projectId, tier, nil
	}

	// 3. 没有 projectId，需要激活
	log.Printf("[antigravity] ResolveProjectId: 没有 projectId，开始激活流程 (tier=%s)...", tier)
	projectId, err = c.OnboardUser(ctx, accessToken, tier)
	if err != nil {
		return "", tier, fmt.Errorf("onboardUser 激活失败: %w", err)
	}

	if projectId == "" {
		return "", tier, fmt.Errorf("onboardUser 完成但未返回 projectId")
	}

	log.Printf("[antigravity] ResolveProjectId: 激活成功, projectId=%s", projectId)
	return projectId, tier, nil
}
