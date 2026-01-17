import api from '../index'

export interface SocialOAuthProvider {
  name: string
  display_name: string
  enabled: boolean
  has_client_id: boolean
  has_client_secret: boolean
  config?: Record<string, unknown>
  created_at: string
  updated_at: string
}

// Response from GetProvider (includes client_id)
export interface SocialOAuthProviderDetail {
  name: string
  display_name: string
  client_id: string
  enabled: boolean
  config?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface UpdateProviderRequest {
  enabled?: boolean
  client_id?: string
  client_secret?: string
  config?: Record<string, unknown>
}

/**
 * Get all OAuth providers for admin
 */
export async function getProviders(): Promise<SocialOAuthProvider[]> {
  const response = await api.get('/api/admin/social-oauth/providers')
  return response.data.data || response.data
}

/**
 * Get a single OAuth provider by name (includes client_id)
 */
export async function getProvider(name: string): Promise<SocialOAuthProviderDetail> {
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
