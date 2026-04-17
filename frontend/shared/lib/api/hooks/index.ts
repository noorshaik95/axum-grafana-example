'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as authApi from '../auth'
import * as coursesApi from '../courses'
import * as assignmentsApi from '../assignments'
import * as gradesApi from '../grades'
import * as usersApi from '../users'
import type {
  LoginRequest,
  RegisterRequest,
  UpdateProfileRequest,
  CourseFilters,
  Course,
} from '../types'

// ---------------------------------------------------------------------------
// Auth hooks
// ---------------------------------------------------------------------------

export function useProfile() {
  return useQuery({
    queryKey: ['profile'],
    queryFn: authApi.getProfile,
    retry: false,
  })
}

export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: LoginRequest) => authApi.login(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile'] })
    },
  })
}

export function useRegister() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: RegisterRequest) => authApi.register(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile'] })
    },
  })
}

export function useLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => authApi.logout(),
    onSuccess: () => {
      queryClient.clear()
    },
  })
}

export function useUpdateProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: UpdateProfileRequest) => authApi.updateProfile(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Course hooks
// ---------------------------------------------------------------------------

export function useCourses(filters?: CourseFilters) {
  return useQuery({
    queryKey: ['courses', filters],
    queryFn: () => coursesApi.listCourses(filters),
  })
}

export function useCourse(id: string) {
  return useQuery({
    queryKey: ['courses', id],
    queryFn: () => coursesApi.getCourse(id),
    enabled: !!id,
  })
}

export function useCreateCourse() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Course>) => coursesApi.createCourse(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['courses'] })
    },
  })
}

export function useEnrollInCourse() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (courseId: string) => coursesApi.enrollInCourse(courseId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['enrollments'] })
      queryClient.invalidateQueries({ queryKey: ['courses'] })
    },
  })
}

export function useMyEnrollments(studentId: string) {
  return useQuery({
    queryKey: ['enrollments', studentId],
    queryFn: () => coursesApi.getMyEnrollments(studentId),
    enabled: !!studentId,
  })
}

// ---------------------------------------------------------------------------
// Assignment hooks
// ---------------------------------------------------------------------------

export function useAssignments(courseId?: string) {
  return useQuery({
    queryKey: ['assignments', courseId],
    queryFn: () => assignmentsApi.listAssignments(courseId),
  })
}

export function useAssignment(id: string) {
  return useQuery({
    queryKey: ['assignments', 'detail', id],
    queryFn: () => assignmentsApi.getAssignment(id),
    enabled: !!id,
  })
}

export function useSubmitAssignment() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (vars: { assignmentId: string; data: { content?: string; fileUrl?: string } }) =>
      assignmentsApi.submitAssignment(vars.assignmentId, vars.data),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({
        queryKey: ['submissions', vars.assignmentId],
      })
      queryClient.invalidateQueries({ queryKey: ['assignments'] })
    },
  })
}

export function useSubmissions(assignmentId: string) {
  return useQuery({
    queryKey: ['submissions', assignmentId],
    queryFn: () => assignmentsApi.getSubmissions(assignmentId),
    enabled: !!assignmentId,
  })
}

// ---------------------------------------------------------------------------
// Grade hooks
// ---------------------------------------------------------------------------

export function useMyGrades(studentId: string) {
  return useQuery({
    queryKey: ['grades', studentId],
    queryFn: () => gradesApi.getMyGrades(studentId),
    enabled: !!studentId,
  })
}

export function useGradebook(courseId: string) {
  return useQuery({
    queryKey: ['gradebook', courseId],
    queryFn: () => gradesApi.getGradebook(courseId),
    enabled: !!courseId,
  })
}

// ---------------------------------------------------------------------------
// User hooks (admin)
// ---------------------------------------------------------------------------

export function useUsers(params?: {
  search?: string
  role?: string
  page?: number
  pageSize?: number
}) {
  return useQuery({
    queryKey: ['users', params],
    queryFn: () => usersApi.listUsers(params),
  })
}

export function useUser(id: string) {
  return useQuery({
    queryKey: ['users', id],
    queryFn: () => usersApi.getUser(id),
    enabled: !!id,
  })
}
