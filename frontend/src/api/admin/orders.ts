/**
 * Admin Payment Orders API endpoints
 * Handles payment order management for admin
 */

import { apiClient } from '../client'
import type { PaymentOrder } from '../payment'

// List orders params
export interface ListOrdersParams {
  page?: number
  page_size?: number
  status?: string
  user_id?: number
  order_type?: string
  search?: string
  start_date?: string
  end_date?: string
}

// List orders response
export interface ListOrdersResponse {
  items: PaymentOrder[]
  total: number
}

// Order stats response
export interface OrderStatsResponse {
  total_orders: number
  total_amount: number
  paid_orders: number
  paid_amount: number
  pending_orders: number
  pending_amount: number
  today_orders: number
  today_amount: number
}

// Order with user info
export interface PaymentOrderWithUser extends PaymentOrder {
  user?: {
    id: number
    email: string
    is_admin: boolean
  }
}

/**
 * List all payment orders (admin)
 */
export async function listOrders(params?: ListOrdersParams): Promise<ListOrdersResponse> {
  const { data } = await apiClient.get<ListOrdersResponse>('/admin/payment-orders', { params })
  return data
}

/**
 * Get order by ID
 */
export async function getOrder(id: number): Promise<PaymentOrderWithUser> {
  const { data } = await apiClient.get<PaymentOrderWithUser>(`/admin/payment-orders/${id}`)
  return data
}

/**
 * Get order statistics
 */
export async function getOrderStats(): Promise<OrderStatsResponse> {
  const { data } = await apiClient.get<OrderStatsResponse>('/admin/payment-orders/stats')
  return data
}

/**
 * Approve an order (for auditing orders)
 */
export async function approveOrder(id: number): Promise<PaymentOrder> {
  const { data } = await apiClient.post<PaymentOrder>(`/admin/payment-orders/${id}/approve`, {})
  return data
}

/**
 * Reject an order
 */
export async function rejectOrder(id: number, reason: string): Promise<PaymentOrder> {
  const { data } = await apiClient.post<PaymentOrder>(`/admin/payment-orders/${id}/reject`, { reason })
  return data
}

/**
 * Manual fulfill an order (for special cases)
 */
export async function fulfillOrder(id: number, tradeNo: string): Promise<PaymentOrder> {
  const { data } = await apiClient.post<PaymentOrder>(`/admin/payment-orders/${id}/fulfill`, { trade_no: tradeNo })
  return data
}

export const adminOrdersAPI = {
  listOrders,
  getOrder,
  getOrderStats,
  approveOrder,
  rejectOrder,
  fulfillOrder
}

export default adminOrdersAPI
