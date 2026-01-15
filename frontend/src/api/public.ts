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

/**
 * 开始 Antigravity OAuth 流程
 */
export async function startAntigravityOAuth(): Promise<OAuthStartResponse> {
  const response = await publicClient.post<{ data: OAuthStartResponse }>(
    '/antigravity/oauth/start'
  )
  return response.data.data
}

/**
 * 完成 Antigravity OAuth 流程
 */
export async function completeAntigravityOAuth(
  request: OAuthCompleteRequest
): Promise<OAuthCompleteResponse> {
  const response = await publicClient.post<{ data: OAuthCompleteResponse }>(
    '/antigravity/oauth/complete',
    request
  )
  return response.data.data
}

export const publicAPI = {
  antigravity: {
    startOAuth: startAntigravityOAuth,
    completeOAuth: completeAntigravityOAuth
  }
}
