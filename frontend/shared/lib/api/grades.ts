import { apiClient } from './client'
import type { Grade, Gradebook } from './types'

export async function getMyGrades(studentId: string): Promise<Grade[]> {
  return apiClient.get<Grade[]>(`/api/students/${studentId}/gradebook`)
}

export async function getGradebook(courseId: string): Promise<Gradebook> {
  return apiClient.get<Gradebook>(`/api/courses/${courseId}/gradebook`)
}

export async function getStudentGradebook(studentId: string): Promise<Gradebook> {
  return apiClient.get<Gradebook>(`/api/students/${studentId}/gradebook`)
}

export async function publishGrade(gradeId: string): Promise<Grade> {
  return apiClient.post<Grade>(`/api/grades/${gradeId}/publish`)
}
