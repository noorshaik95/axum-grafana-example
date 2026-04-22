import { apiClient } from '../../../shared/lib/api/client'

// W5 discussion-service — gateway routes live under /api/discussions/*.

export interface DiscussionThread {
  id: string
  course_id: string
  title: string
  created_by: string
  created_at: string
  last_activity_at: string
  reply_count: number
}

export interface ListThreadsResponse {
  threads: DiscussionThread[]
  next_cursor?: string
  has_more: boolean
}

export interface CreateThreadDto {
  course_id: string
  title: string
  initial_post: string
}

export async function listThreads(params: {
  courseId?: string
  limit?: number
  cursor?: string
}): Promise<ListThreadsResponse> {
  const q = new URLSearchParams()
  if (params.courseId) q.set('course_id', params.courseId)
  if (params.limit) q.set('limit', String(params.limit))
  if (params.cursor) q.set('cursor', params.cursor)
  const suffix = q.toString() ? `?${q.toString()}` : ''
  return apiClient.get<ListThreadsResponse>(`/api/discussions/threads${suffix}`)
}

export async function createThread(data: CreateThreadDto): Promise<DiscussionThread> {
  return apiClient.post<DiscussionThread>('/api/discussions/threads', data)
}
