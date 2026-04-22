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

export async function lockCourse(id: string): Promise<void> {
  return apiClient.patch(`/api/courses/${id}/lock`)
}

export async function unlockCourse(id: string): Promise<void> {
  return apiClient.patch(`/api/courses/${id}/unlock`)
}

export async function getModules(courseId: string): Promise<Module[]> {
  return apiClient.get<Module[]>(`/api/courses/${courseId}/modules`)
}

export async function createModule(courseId: string, data: CreateModuleDto): Promise<Module> {
  return apiClient.post<Module>(`/api/courses/${courseId}/modules`, data)
}

export async function updateModule(
  courseId: string,
  moduleId: string,
  data: UpdateModuleDto
): Promise<Module> {
  return apiClient.put<Module>(`/api/courses/${courseId}/modules/${moduleId}`, data)
}

export async function deleteModule(courseId: string, moduleId: string): Promise<void> {
  return apiClient.delete(`/api/courses/${courseId}/modules/${moduleId}`)
}

export async function reorderModules(courseId: string, order: ModuleOrder[]): Promise<void> {
  return apiClient.patch(`/api/courses/${courseId}/modules/reorder`, order)
}

export async function getLessons(courseId: string, moduleId: string): Promise<Lesson[]> {
  return apiClient.get<Lesson[]>(`/api/courses/${courseId}/modules/${moduleId}/lessons`)
}

export async function createLesson(
  courseId: string,
  moduleId: string,
  data: CreateLessonDto
): Promise<Lesson> {
  return apiClient.post<Lesson>(`/api/courses/${courseId}/modules/${moduleId}/lessons`, data)
}

export async function updateLesson(
  courseId: string,
  moduleId: string,
  lessonId: string,
  data: UpdateLessonDto
): Promise<Lesson> {
  return apiClient.put<Lesson>(
    `/api/courses/${courseId}/modules/${moduleId}/lessons/${lessonId}`,
    data
  )
}

export async function deleteLesson(
  courseId: string,
  moduleId: string,
  lessonId: string
): Promise<void> {
  return apiClient.delete(`/api/courses/${courseId}/modules/${moduleId}/lessons/${lessonId}`)
}

export async function reorderLessons(
  courseId: string,
  moduleId: string,
  order: LessonOrder[]
): Promise<void> {
  return apiClient.patch(`/api/courses/${courseId}/modules/${moduleId}/lessons/reorder`, order)
}

export async function getCourseAnalytics(courseId: string): Promise<CourseAnalytics> {
  return apiClient.get<CourseAnalytics>(`/api/courses/${courseId}/analytics`)
}

export async function getCourseRoster(courseId: string): Promise<CourseRosterResponse> {
  return apiClient.get<CourseRosterResponse>(`/api/courses/${courseId}/roster`)
}
