import { apiClient } from '../../../shared/lib/api/client'

export interface Announcement {
  id: string
  title: string
  content: string
  courseId: string | null
  courseName?: string
  authorId: string
  authorName?: string
  publishedAt: string | null
  scheduledAt: string | null
  status: 'draft' | 'published' | 'scheduled'
  createdAt: string
  updatedAt: string
}

export interface CreateAnnouncementDto {
  title: string
  content: string
  courseId?: string
  scheduledAt?: string
  publish?: boolean
}

export interface UpdateAnnouncementDto {
  title?: string
  content?: string
  courseId?: string
  scheduledAt?: string
}

export async function listAnnouncements(courseId?: string): Promise<Announcement[]> {
  const params = courseId ? `?courseId=${courseId}` : ''
  return apiClient.get<Announcement[]>(`/api/announcements${params}`)
}

export async function getAnnouncement(id: string): Promise<Announcement> {
  return apiClient.get<Announcement>(`/api/announcements/${id}`)
}

export async function createAnnouncement(data: CreateAnnouncementDto): Promise<Announcement> {
  return apiClient.post<Announcement>('/api/announcements', data)
}

export async function updateAnnouncement(
  id: string,
  data: UpdateAnnouncementDto
): Promise<Announcement> {
  return apiClient.put<Announcement>(`/api/announcements/${id}`, data)
}

export async function publishAnnouncement(id: string): Promise<Announcement> {
  return apiClient.post<Announcement>(`/api/announcements/${id}/publish`)
}

export async function deleteAnnouncement(id: string): Promise<void> {
  return apiClient.delete(`/api/announcements/${id}`)
}
