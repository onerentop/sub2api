/**
 * Admin Users API endpoints
 * Handles user management for administrators
 */

import { apiClient } from '../client'
import type { AdminUser, UpdateUserRequest, PaginatedResponse } from '@/types'

/**
 * List all users with pagination
 * @param page - Page number (default: 1)
 * @param pageSize - Items per page (default: 20)
 * @param filters - Optional filters (status, role, search, attributes)
 * @param options - Optional request options (signal)
 * @returns Paginated list of users
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: 'active' | 'disabled'
    role?: 'admin' | 'user'
    search?: string
    attributes?: Record<number, string>  // attributeId -> value
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<PaginatedResponse<AdminUser>> {
  // Build params with attribute filters in attr[id]=value format
  const params: Record<string, any> = {
    page,
    page_size: pageSize,
    status: filters?.status,
    role: filters?.role,
    search: filters?.search
  }

  // Add attribute filters as attr[id]=value
  if (filters?.attributes) {
    for (const [attrId, value] of Object.entries(filters.attributes)) {
      if (value) {
        params[`attr[${attrId}]`] = value
      }
    }
  }
  const { data } = await apiClient.get<PaginatedResponse<AdminUser>>('/admin/users', {
    params,
    signal: options?.signal
  })
  return data
}

/**
 * Get user by ID
 * @param id - User ID
 * @returns User details
 */
export async function getById(id: number): Promise<AdminUser> {
  const { data } = await apiClient.get<AdminUser>(`/admin/users/${id}`)
  return data
}

/**
 * Create new user
 * @param userData - User data (email, password, etc.)
 * @returns Created user
 */
export async function create(userData: {
  email: string
  password: string
  balance?: number
  concurrency?: number
  allowed_groups?: number[] | null
}): Promise<AdminUser> {
  const { data } = await apiClient.post<AdminUser>('/admin/users', userData)
  return data
}

/**
 * Update user
 * @param id - User ID
 * @param updates - Fields to update
 * @returns Updated user
 */
export async function update(id: number, updates: UpdateUserRequest): Promise<AdminUser> {
  const { data } = await apiClient.put<AdminUser>(`/admin/users/${id}`, updates)
  return data
}

/**
 * Delete user
 * @param id - User ID
 * @returns Success confirmation
 */
export async function deleteUser(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/users/${id}`)
  return data
}

/**
 * Update user balance
 * @param id - User ID
 * @param balance - New balance
 * @param operation - Operation type ('set', 'add', 'subtract')
 * @param notes - Optional notes for the balance adjustment
 * @returns Updated user
 */
export async function updateBalance(
  id: number,
  balance: number,
  operation: 'set' | 'add' | 'subtract' = 'set',
  notes?: string
): Promise<AdminUser> {
  const { data } = await apiClient.post<AdminUser>(`/admin/users/${id}/balance`, {
    balance,
    operation,
    notes: notes || ''
  })
  return data
}

/**
 * Update user concurrency
 * @param id - User ID
 * @param concurrency - New concurrency limit
 * @returns Updated user
 */
export async function updateConcurrency(id: number, concurrency: number): Promise<AdminUser> {
  return update(id, { concurrency })
}

/**
 * Toggle user status
 * @param id - User ID
 * @param status - New status
 * @returns Updated user
 */
export async function toggleStatus(id: number, status: 'active' | 'disabled'): Promise<AdminUser> {
  return update(id, { status })
}

/**
 * Get user's API keys
 * @param id - User ID
 * @returns List of user's API keys
 */
export async function getUserApiKeys(id: number): Promise<PaginatedResponse<any>> {
  const { data } = await apiClient.get<PaginatedResponse<any>>(`/admin/users/${id}/api-keys`)
  return data
}

/**
 * Get user's usage statistics
 * @param id - User ID
 * @param period - Time period
 * @returns User usage statistics
 */
export async function getUserUsageStats(
  id: number,
  period: string = 'month'
): Promise<{
  total_requests: number
  total_cost: number
  total_tokens: number
}> {
  const { data } = await apiClient.get<{
    total_requests: number
    total_cost: number
    total_tokens: number
  }>(`/admin/users/${id}/usage`, {
    params: { period }
  })
  return data
}

/**
 * Bulk update parameters for multiple users
 */
export interface BulkUpdateUsersParams {
  status?: 'active' | 'disabled'
  concurrency?: number
  allowed_groups?: number[]
  balance_daily_quota?: number
  balance_weekly_quota?: number
  balance_adjustment?: number  // 正数增加，负数减少
}

/**
 * Bulk update result
 */
export interface BulkUpdateUsersResult {
  success: number
  failed: number
  success_ids: number[]
  failed_ids: number[]
  results: Array<{
    user_id: number
    success: boolean
    error?: string
  }>
}

/**
 * Batch delete result
 */
export interface BatchDeleteUsersResult {
  success: number
  failed: number
  success_ids: number[]
  failed_ids: number[]
}

/**
 * Balance history item
 */
export interface BalanceHistoryItem {
  id: number
  user_id: number
  type: 'balance' | 'admin_balance' | 'concurrency' | 'admin_concurrency' | 'subscription'
  value: number
  balance_before: number
  balance_after: number
  notes: string
  operator_id: number | null
  created_at: string
  used_at?: string
  code?: string
  validity_days?: number
  group?: { id: number; name: string } | null
}

/**
 * Bulk update multiple users
 * @param userIds - Array of user IDs
 * @param updates - Fields to update
 * @returns Bulk update result
 */
export async function bulkUpdate(
  userIds: number[],
  updates: BulkUpdateUsersParams
): Promise<BulkUpdateUsersResult> {
  const { data } = await apiClient.post<BulkUpdateUsersResult>('/admin/users/bulk-update', {
    user_ids: userIds,
    ...updates
  })
  return data
}

/**
 * Batch delete multiple users
 * @param userIds - Array of user IDs to delete
 * @returns Batch delete result
 */
export async function batchDelete(userIds: number[]): Promise<BatchDeleteUsersResult> {
  const { data } = await apiClient.post<BatchDeleteUsersResult>('/admin/users/batch-delete', {
    user_ids: userIds
  })
  return data
}

/**
 * Balance history response
 */
export interface BalanceHistoryResponse {
  items: BalanceHistoryItem[]
  total: number
  total_recharged: number
}

/**
 * Get user balance history
 * @param id - User ID
 * @param page - Page number
 * @param pageSize - Items per page
 * @param type - Optional type filter
 * @returns Paginated balance history
 */
export async function getUserBalanceHistory(
  id: number,
  page: number = 1,
  pageSize: number = 15,
  type?: string
): Promise<BalanceHistoryResponse> {
  const { data } = await apiClient.get<BalanceHistoryResponse>(`/admin/users/${id}/balance-history`, {
    params: { page, page_size: pageSize, type }
  })
  return data
}

export const usersAPI = {
  list,
  getById,
  create,
  update,
  delete: deleteUser,
  updateBalance,
  updateConcurrency,
  toggleStatus,
  getUserApiKeys,
  getUserUsageStats,
  bulkUpdate,
  batchDelete,
  getUserBalanceHistory
}

export default usersAPI
