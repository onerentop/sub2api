import api from './index'

// 社交登录提供商公开信息
export interface SocialProvider {
  name: string
  display_name: string
  enabled: boolean
}

// 启动 OAuth 结果
export interface StartOAuthResult {
  auth_url: string
  session_id: string
}

// 回调处理结果
export interface HandleCallbackResult {
  token: string
  redirect_to: string
  is_new_user: boolean
}

// 用户绑定信息
export interface UserOAuthBinding {
  provider: string
  provider_email?: string
  provider_username?: string
  provider_avatar?: string
  created_at: string
}

// 获取可用的社交登录提供商
export async function getSocialProviders(): Promise<SocialProvider[]> {
  const response = await api.get<SocialProvider[]>('/social/providers')
  return response.data
}

// 启动社交登录
export async function startSocialLogin(
  provider: string,
  redirectTo?: string
): Promise<StartOAuthResult> {
  const response = await api.post<StartOAuthResult>('/social/login/start', {
    provider,
    redirect_to: redirectTo || '/dashboard'
  })
  return response.data
}

// 处理社交登录回调
export async function handleSocialLoginCallback(
  provider: string,
  code: string,
  state: string,
  sessionId: string
): Promise<HandleCallbackResult> {
  const response = await api.post<HandleCallbackResult>('/social/login/callback', {
    provider,
    code,
    state,
    session_id: sessionId
  })
  return response.data
}

// 获取用户绑定列表
export async function getUserBindings(): Promise<UserOAuthBinding[]> {
  const response = await api.get<UserOAuthBinding[]>('/social/bindings')
  return response.data
}

// 启动社交账号绑定
export async function startSocialBind(provider: string): Promise<StartOAuthResult> {
  const response = await api.post<StartOAuthResult>('/social/bind/start', {
    provider
  })
  return response.data
}

// 处理社交账号绑定回调
export async function handleSocialBindCallback(
  provider: string,
  code: string,
  state: string,
  sessionId: string
): Promise<void> {
  await api.post('/social/bind/callback', {
    provider,
    code,
    state,
    session_id: sessionId
  })
}

// 解绑社交账号
export async function unbindSocialAccount(provider: string): Promise<void> {
  await api.post('/social/unbind', {
    provider
  })
}
