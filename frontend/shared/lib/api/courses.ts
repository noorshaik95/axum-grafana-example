import { apiClient } from './client'
import type { Course, CourseFilters, Enrollment, PaginatedResponse } from './types'

export async function listCourses(filters?: CourseFilters): Promise<PaginatedResponse<Course>> {
  const params = new URLSearchParams()
  if (filters?.search) params.set('search', filters.search)
  if (filters?.term) params.set('term', filters.term)
  if (filters?.instructorId) params.set('instructorId', filters.instructorId)
  if (filters?.isPublished !== undefined) params.set('isPublished', String(filters.isPublished))
  if (filters?.department) params.set('department', filters.department)
  if (filters?.page) params.set('page', String(filters.page))
  if (filters?.pageSize) params.set('pageSize', String(filters.pageSize))
  const query = params.toString()
  return apiClient.get<PaginatedResponse<Course>>(`/api/courses${query ? `?${query}` : ''}`)
}

export async function getCourse(id: string): Promise<Course> {
  return apiClient.get<Course>(`/api/courses/${id}`)
}

export async function createCourse(data: Partial<Course>): Promise<Course> {
  return apiClient.post<Course>('/api/courses', data)
}

export async function updateCourse(id: string, data: Partial<Course>): Promise<Course> {
  return apiClient.put<Course>(`/api/courses/${id}`, data)
}

export async function deleteCourse(id: string): Promise<void> {
  return apiClient.delete(`/api/courses/${id}`)
}

export async function enrollInCourse(courseId: string): Promise<Enrollment> {
  return apiClient.post<Enrollment>(`/api/courses/${courseId}/enroll`)
}

export async function getMyEnrollments(studentId: string): Promise<Enrollment[]> {
  return apiClient.get<Enrollment[]>(`/api/students/${studentId}/enrollments`)
}
