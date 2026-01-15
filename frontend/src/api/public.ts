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

export const publicAPI = {
  antigravity: {
    startOAuth: startAntigravityOAuth,
    completeOAuth: completeAntigravityOAuth
  }
}
