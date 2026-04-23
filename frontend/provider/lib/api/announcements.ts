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

// T2 reject-with-toast: /api/announcements write ops have no gateway routes
//   + no corresponding service yet. Thrown errors surface as UI toasts.
//   Flip back to apiClient.* when an announcements service lands post-MVP.

export async function createAnnouncement(_data: CreateAnnouncementDto): Promise<Announcement> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function updateAnnouncement(
  _id: string,
  _data: UpdateAnnouncementDto
): Promise<Announcement> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function publishAnnouncement(_id: string): Promise<Announcement> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function deleteAnnouncement(_id: string): Promise<void> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}
