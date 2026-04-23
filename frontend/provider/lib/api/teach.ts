import { apiClient } from '../../../shared/lib/api/client'

// Endpoints consumed by the /teach landing card (W8 + W5 stub contracts).

export interface NextLecture {
  course_id: string
  course_title: string
  lecture_id?: string
  topic: string
  starts_at: string // ISO
  ends_at: string // ISO
  location?: string
  live: boolean
}

export interface DiscussionsNeedingReplyCount {
  count: number
  oldest_hours?: number
}

export async function getNextLecture(): Promise<NextLecture | null> {
  return apiClient.get<NextLecture | null>('/api/courses/next-lecture')
}

export async function getDiscussionsNeedingReplyCount(): Promise<DiscussionsNeedingReplyCount> {
  // W5.4 — uses limit=1 so the server can skip expensive pagination.
  return apiClient.get<DiscussionsNeedingReplyCount>('/api/discussions/needing-reply?limit=1')
}
