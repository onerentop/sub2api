/**
 * Admin Products API endpoints
 * Handles product CRUD operations for admin
 */

import { apiClient } from '../client'
import type { Product } from '../payment'

// Create product request
export interface CreateProductRequest {
  name: string
  description?: string
  type: 'balance' | 'subscription'
  price_cny: number
  value: number
  group_id?: number
  is_active?: boolean
  sort_order?: number
}

// Update product request
export interface UpdateProductRequest {
  name?: string
  description?: string
  type?: 'balance' | 'subscription'
  price_cny?: number
  value?: number
  group_id?: number | null
  is_active?: boolean
  sort_order?: number
}

// List products params
export interface ListProductsParams {
  page?: number
  page_size?: number
  type?: string
  is_active?: boolean
  search?: string
}

// List products response
export interface ListProductsResponse {
  items: Product[]
  total: number
}

/**
 * List all products (admin)
 */
export async function listProducts(params?: ListProductsParams): Promise<ListProductsResponse> {
  const { data } = await apiClient.get<ListProductsResponse>('/admin/products', { params })
  return data
}

/**
 * Get product by ID
 */
export async function getProduct(id: number): Promise<Product> {
  const { data } = await apiClient.get<Product>(`/admin/products/${id}`)
  return data
}

/**
 * Create a new product
 */
export async function createProduct(request: CreateProductRequest): Promise<Product> {
  const { data } = await apiClient.post<Product>('/admin/products', request)
  return data
}

/**
 * Update a product
 */
export async function updateProduct(id: number, request: UpdateProductRequest): Promise<Product> {
  const { data } = await apiClient.put<Product>(`/admin/products/${id}`, request)
  return data
}

/**
 * Delete a product (soft delete)
 */
export async function deleteProduct(id: number): Promise<void> {
  await apiClient.delete(`/admin/products/${id}`)
}

/**
 * Toggle product active status
 */
export async function toggleProductActive(id: number): Promise<Product> {
  const { data } = await apiClient.post<Product>(`/admin/products/${id}/toggle-active`)
  return data
}

export const adminProductsAPI = {
  listProducts,
  getProduct,
  createProduct,
  updateProduct,
  deleteProduct,
  toggleProductActive
}

export default adminProductsAPI
