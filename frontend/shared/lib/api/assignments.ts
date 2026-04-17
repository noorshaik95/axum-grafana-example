import { apiClient } from './client'
import type { Assignment, Submission, PaginatedResponse } from './types'

export async function listAssignments(courseId?: string): Promise<PaginatedResponse<Assignment>> {
  const params = courseId ? `?courseId=${courseId}` : ''
  return apiClient.get<PaginatedResponse<Assignment>>(`/api/assignments${params}`)
}

export async function getAssignment(id: string): Promise<Assignment> {
  return apiClient.get<Assignment>(`/api/assignments/${id}`)
}

export async function submitAssignment(
  assignmentId: string,
  data: { content?: string; fileUrl?: string }
): Promise<Submission> {
  return apiClient.post<Submission>(`/api/assignments/${assignmentId}/submissions`, data)
}

export async function getSubmissions(assignmentId: string): Promise<Submission[]> {
  return apiClient.get<Submission[]>(`/api/assignments/${assignmentId}/submissions`)
}
