/**
 * Public API Client
 * 公开接口（无需认证）
 */

import axios from 'axios'

// 创建独立的 axios 实例，不带认证
const publicClient = axios.create({
  baseURL: '/public',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// ==================== Antigravity OAuth ====================

export interface OAuthStartResponse {
  auth_url: string
  session_id: string
  state: string
}

export interface OAuthCompleteRequest {
  session_id: string
  state: string
  code: string
}

export interface OAuthCompleteResponse {
  success: boolean
  message: string
  email?: string
  is_new: boolean
  session_id?: string // 用于后续唤醒请求
}

export interface WakeRequest {
  session_id: string
  models?: string[] // 可选，默认使用 gemini-3-flash
  custom_prompt?: string // 自定义提示词
  max_output_tokens?: number // 最大输出 token 数
}

// 单个模型的唤醒结果
export interface WakeModelResult {
  model: string
  success: boolean
  message?: string
  text?: string
  duration?: number // 耗时 (ms)
  prompt_tokens?: number
  completion_tokens?: number
  total_tokens?: number
  trace_id?: string
}

export interface WakeResponse {
  success: boolean
  message?: string
  model: string
  text?: string
  duration?: number // 耗时 (ms)
  // 多模型结果
  results?: WakeModelResult[]
  // Token 统计（聚合）
  prompt_tokens?: number
  completion_tokens?: number
  total_tokens?: number
}

interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

/**
 * 开始 Antigravity OAuth 流程
 */
export async function startAntigravityOAuth(): Promise<OAuthStartResponse> {
  const response = await publicClient.post<ApiResponse<OAuthStartResponse>>(
    '/antigravity/oauth/start'
  )
  // 检查响应结构
  if (response.data && response.data.data) {
    return response.data.data
  }
  // 兼容直接返回数据的情况
  if (response.data && (response.data as unknown as OAuthStartResponse).auth_url) {
    return response.data as unknown as OAuthStartResponse
  }
  throw new Error('Invalid response format')
}

/**
 * 完成 Antigravity OAuth 流程
 */
export async function completeAntigravityOAuth(
  request: OAuthCompleteRequest
): Promise<OAuthCompleteResponse> {
  const response = await publicClient.post<ApiResponse<OAuthCompleteResponse>>(
    '/antigravity/oauth/complete',
    request
  )
  // 检查响应结构
  if (response.data && response.data.data) {
    return response.data.data
  }
  // 兼容直接返回数据的情况
  if (response.data && (response.data as unknown as OAuthCompleteResponse).success !== undefined) {
    return response.data as unknown as OAuthCompleteResponse
  }
  throw new Error('Invalid response format')
}

/**
 * 执行唤醒测试（触发配额）
 */
export async function wakeAntigravity(request: WakeRequest): Promise<WakeResponse> {
  const response = await publicClient.post<ApiResponse<WakeResponse>>(
    '/antigravity/wake',
    request
  )
  // 检查响应结构
  if (response.data && response.data.data) {
    return response.data.data
  }
  // 兼容直接返回数据的情况
  if (response.data && (response.data as unknown as WakeResponse).success !== undefined) {
    return response.data as unknown as WakeResponse
  }
  throw new Error('Invalid response format')
}

export const publicAPI = {
  antigravity: {
    startOAuth: startAntigravityOAuth,
    completeOAuth: completeAntigravityOAuth,
    wake: wakeAntigravity
  }
}
