/**
 * Public Announcements API endpoints
 * Used by the frontend to display active announcements
 */

import { apiClient } from './client'
import type { ActiveAnnouncementsResponse } from '@/types'

/**
 * Get currently active announcements for display
 * This endpoint is public and does not require authentication
 */
export async function getActiveAnnouncements(): Promise<ActiveAnnouncementsResponse> {
  const { data } = await apiClient.get<ActiveAnnouncementsResponse>('/announcements/active')
  return data
}

const announcementsAPI = {
  getActiveAnnouncements
}

export default announcementsAPI
