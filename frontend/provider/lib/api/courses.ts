import { apiClient } from '../../../shared/lib/api/client'
import type {
  Course,
  CourseFilters,
  Module,
  Lesson,
  PaginatedResponse,
} from '../../../shared/lib/api/types'

export interface ModuleOrder {
  moduleId: string
  displayOrder: number
}

export interface LessonOrder {
  lessonId: string
  displayOrder: number
}

export interface CreateModuleDto {
  name: string
  description?: string
}

export interface UpdateModuleDto {
  name?: string
  description?: string
}

export interface CreateLessonDto {
  name: string
  description?: string
}

export interface CourseAnalytics {
  course_id: string
  engagement_pct: number
  completion_pct: number
  median_grade?: number | null
  at_risk_count: number
  updated_at: string
}

export type EnrollmentStatus = 'active' | 'dropped' | 'completed' | 'waitlisted'

export interface RosterEnrollment {
  id: string
  course_id: string
  student_id: string
  student_name?: string
  status: EnrollmentStatus
  enrolled_at: string
  section_id?: string
}

export interface CourseRosterResponse {
  enrollments: RosterEnrollment[]
  total_count: number
}

export interface UpdateLessonDto {
  name?: string
  description?: string
  published?: boolean
  availableFrom?: string
}

export async function listCourses(filters?: CourseFilters): Promise<PaginatedResponse<Course>> {
  const params = new URLSearchParams()
  if (filters?.search) params.set('search', filters.search)
  if (filters?.term) params.set('term', filters.term)
  if (filters?.instructorId) params.set('instructorId', filters.instructorId)
  if (filters?.isPublished !== undefined) params.set('isPublished', String(filters.isPublished))
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

// T2 reject-with-toast: the remaining course/module/lesson write ops below
//   have no gateway routes + no corresponding course-service RPCs today.
//   We throw client-side so mutations surface as toasts rather than 404s.
//   Flip back to apiClient calls when course-expert exposes them (post-MVP).

export async function lockCourse(_id: string): Promise<void> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function unlockCourse(_id: string): Promise<void> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function getModules(courseId: string): Promise<Module[]> {
  return apiClient.get<Module[]>(`/api/courses/${courseId}/modules`)
}

export async function createModule(courseId: string, data: CreateModuleDto): Promise<Module> {
  return apiClient.post<Module>(`/api/courses/${courseId}/modules`, data)
}

export async function updateModule(
  _courseId: string,
  _moduleId: string,
  _data: UpdateModuleDto
): Promise<Module> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function deleteModule(_courseId: string, _moduleId: string): Promise<void> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function reorderModules(_courseId: string, _order: ModuleOrder[]): Promise<void> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function getLessons(courseId: string, moduleId: string): Promise<Lesson[]> {
  return apiClient.get<Lesson[]>(`/api/courses/${courseId}/modules/${moduleId}/lessons`)
}

export async function createLesson(
  _courseId: string,
  _moduleId: string,
  _data: CreateLessonDto
): Promise<Lesson> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function updateLesson(
  _courseId: string,
  _moduleId: string,
  _lessonId: string,
  _data: UpdateLessonDto
): Promise<Lesson> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function deleteLesson(
  _courseId: string,
  _moduleId: string,
  _lessonId: string
): Promise<void> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function reorderLessons(
  _courseId: string,
  _moduleId: string,
  _order: LessonOrder[]
): Promise<void> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}

export async function getCourseAnalytics(courseId: string): Promise<CourseAnalytics> {
  return apiClient.get<CourseAnalytics>(`/api/courses/${courseId}/analytics`)
}

export async function getCourseRoster(courseId: string): Promise<CourseRosterResponse> {
  return apiClient.get<CourseRosterResponse>(`/api/courses/${courseId}/roster`)
}
