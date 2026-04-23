import { apiClient } from '../../../shared/lib/api/client'

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

export async function cancelSession(_id: string): Promise<void> {
  // T2: PATCH /api/video/sessions/:id/cancel has no gateway route + no
  //   corresponding video-service RPC. Reject client-side so the UI surfaces
  //   a toast instead of a 404. Remove this stub when the cancel endpoint
  //   lands (video-expert follow-up, post-MVP).
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

// W11.2 Lecture Q&A

export interface LectureQuestion {
  id: string
  user_id: string
  text: string
  upvotes: number
  submitted_at_unix: number
}

export interface GetLectureQuestionsResponse {
  questions: LectureQuestion[]
}

export async function listLectureQuestions(
  lectureId: string,
  limit?: number
): Promise<GetLectureQuestionsResponse> {
  const q = new URLSearchParams()
  if (limit) q.set('limit', String(limit))
  const suffix = q.toString() ? `?${q.toString()}` : ''
  return apiClient.get<GetLectureQuestionsResponse>(`/api/video/lectures/${lectureId}/qa${suffix}`)
}

export async function upvoteLectureQuestion(
  lectureId: string,
  questionId: string
): Promise<LectureQuestion> {
  return apiClient.post<LectureQuestion>(
    `/api/video/lectures/${lectureId}/qa/${questionId}/upvote`,
    {}
  )
}
