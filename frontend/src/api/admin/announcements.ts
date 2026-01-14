/**
 * Admin Announcements API endpoints
 */

import { apiClient } from '../client'
import type {
  Announcement,
  CreateAnnouncementRequest,
  UpdateAnnouncementRequest,
  AnnouncementSortItem,
  BasePaginationResponse
} from '@/types'

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    enabled?: boolean
    type?: string
  }
): Promise<BasePaginationResponse<Announcement>> {
  const { data } = await apiClient.get<BasePaginationResponse<Announcement>>('/admin/announcements', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function getById(id: number): Promise<Announcement> {
  const { data } = await apiClient.get<Announcement>(`/admin/announcements/${id}`)
  return data
}

export async function create(request: CreateAnnouncementRequest): Promise<Announcement> {
  const { data } = await apiClient.post<Announcement>('/admin/announcements', request)
  return data
}

export async function update(id: number, request: UpdateAnnouncementRequest): Promise<Announcement> {
  const { data } = await apiClient.put<Announcement>(`/admin/announcements/${id}`, request)
  return data
}

export async function deleteAnnouncement(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/announcements/${id}`)
  return data
}

export async function updateSortOrders(items: AnnouncementSortItem[]): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>('/admin/announcements/sort', { items })
  return data
}

const announcementsAPI = {
  list,
  getById,
  create,
  update,
  delete: deleteAnnouncement,
  updateSortOrders
}

export default announcementsAPI
