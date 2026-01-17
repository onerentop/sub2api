/**
 * Payment API endpoints
 * Handles product listing, order creation, and payment operations
 */

import { apiClient } from './client'

// Product types
export interface Product {
  id: number
  name: string
  description?: string
  type: 'balance' | 'subscription'
  price_cny: number
  value: number
  group_id?: number
  is_active: boolean
  sort_order: number
  created_at: string
  updated_at: string
  group?: {
    id: number
    name: string
  }
}

// Payment order types
export interface PaymentOrder {
  id: number
  user_id: number
  product_id?: number
  order_no: string
  trade_no?: string
  amount_cny: number
  amount_value: number
  order_type: 'balance' | 'subscription'
  payment_method: 'wechat' | 'alipay'
  status: 'pending' | 'paid' | 'failed' | 'refunded' | 'auditing'
  paid_at?: string
  remark?: string
  created_at: string
  updated_at: string
  product?: Product
}

// Create order request
export interface CreateOrderRequest {
  product_id?: number
  custom_amount?: number
  payment_method: 'wechat' | 'alipay'
}

// Create order response
export interface CreateOrderResponse {
  order_no: string
  payment_url: string
  amount_cny: number
  amount_value: number
  order_type: string
}

// Payment config response
export interface PaymentConfigResponse {
  enabled: boolean
  min_amount: number
  max_amount: number
  audit_threshold: number
  payment_methods: string[]
  cny_usd_rate: number
}

/**
 * Get active products list (for user purchase)
 */
export async function getProducts(): Promise<Product[]> {
  const { data } = await apiClient.get<Product[]>('/payment/products')
  return data
}

/**
 * Create a payment order
 */
export async function createOrder(request: CreateOrderRequest): Promise<CreateOrderResponse> {
  const { data } = await apiClient.post<CreateOrderResponse>('/payment/orders', request)
  return data
}

/**
 * Get order status by order number
 */
export async function getOrderStatus(orderNo: string): Promise<PaymentOrder> {
  const { data } = await apiClient.get<PaymentOrder>(`/payment/orders/${orderNo}`)
  return data
}

/**
 * Get user's payment orders history
 */
export async function getOrderHistory(params?: {
  page?: number
  page_size?: number
  status?: string
}): Promise<{ items: PaymentOrder[]; total: number }> {
  const { data } = await apiClient.get<{ items: PaymentOrder[]; total: number }>(
    '/payment/orders',
    { params }
  )
  return data
}

/**
 * Get payment configuration
 */
export async function getPaymentConfig(): Promise<PaymentConfigResponse> {
  const { data } = await apiClient.get<PaymentConfigResponse>('/payment/config')
  return data
}

export const paymentAPI = {
  getProducts,
  createOrder,
  getOrderStatus,
  getOrderHistory,
  getPaymentConfig
}

export default paymentAPI
