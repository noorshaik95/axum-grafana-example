import { apiClient } from '../../../../shared/lib/api/client'

export interface VideoSession {
  id: string
  title: string
  courseId: string
  courseName?: string
  hostId: string
  scheduledAt: string
  duration: number
  joinUrl?: string
  recordingUrl?: string
  status: 'scheduled' | 'live' | 'ended' | 'cancelled'
  invitedStudentIds: string[]
  createdAt: string
  updatedAt: string
}

export interface CreateSessionDto {
  title: string
  courseId: string
  scheduledAt: string
  duration: number
  invitedStudentIds?: string[]
}

export interface UpdateSessionDto {
  title?: string
  scheduledAt?: string
  duration?: number
  invitedStudentIds?: string[]
}

export async function listSessions(params?: {
  courseId?: string
  status?: string
}): Promise<VideoSession[]> {
  const searchParams = new URLSearchParams()
  if (params?.courseId) searchParams.set('courseId', params.courseId)
  if (params?.status) searchParams.set('status', params.status)
  const query = searchParams.toString()
  return apiClient.get<VideoSession[]>(`/api/video/sessions${query ? `?${query}` : ''}`)
}

export async function getSession(id: string): Promise<VideoSession> {
  return apiClient.get<VideoSession>(`/api/video/sessions/${id}`)
}

export async function createSession(data: CreateSessionDto): Promise<VideoSession> {
  return apiClient.post<VideoSession>('/api/video/sessions', data)
}

export async function updateSession(id: string, data: UpdateSessionDto): Promise<VideoSession> {
  return apiClient.put<VideoSession>(`/api/video/sessions/${id}`, data)
}

export async function cancelSession(id: string): Promise<void> {
  return apiClient.patch(`/api/video/sessions/${id}/cancel`)
}
