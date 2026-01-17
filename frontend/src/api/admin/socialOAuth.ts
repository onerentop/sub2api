import api from '../index'

export interface SocialOAuthProvider {
  id: number
  name: string
  display_name: string
  enabled: boolean
  client_id: string
  client_secret_set: boolean  // Whether secret is set (not exposed)
  redirect_uri: string
  scopes: string[]
  extra_config: Record<string, string>
  created_at: string
  updated_at: string
}

export interface UpdateProviderRequest {
  display_name?: string
  enabled?: boolean
  client_id?: string
  client_secret?: string
  redirect_uri?: string
  scopes?: string[]
  extra_config?: Record<string, string>
}

/**
 * Get all OAuth providers for admin
 */
export async function getProviders(): Promise<SocialOAuthProvider[]> {
  const response = await api.get('/api/admin/social-oauth/providers')
  return response.data.data || response.data
}

/**
 * Get a single OAuth provider by name
 */
export async function getProvider(name: string): Promise<SocialOAuthProvider> {
  const response = await api.get(`/api/admin/social-oauth/providers/${name}`)
  return response.data.data || response.data
}

/**
 * Update an OAuth provider
 */
export async function updateProvider(name: string, data: UpdateProviderRequest): Promise<SocialOAuthProvider> {
  const response = await api.put(`/api/admin/social-oauth/providers/${name}`, data)
  return response.data.data || response.data
}
