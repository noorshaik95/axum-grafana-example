'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as coursesApi from './courses'
import * as gradingApi from './grading'
import * as videoApi from './video'
import * as announcementsApi from './announcements'
import * as messagesApi from './messages'
import * as discussionApi from './discussion'
import type { Course, CourseFilters } from '../../../shared/lib/api/types'

// -- Courses --

export function useInstructorCourses(filters?: CourseFilters) {
  return useQuery({
    queryKey: ['provider-courses', filters],
    queryFn: () => coursesApi.listCourses(filters),
  })
}

export function useInstructorCourse(id: string) {
  return useQuery({
    queryKey: ['provider-courses', id],
    queryFn: () => coursesApi.getCourse(id),
    enabled: !!id,
  })
}

export function useCreateCourse() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Course>) => coursesApi.createCourse(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['provider-courses'] }),
  })
}

export function useUpdateCourse() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Course> }) =>
      coursesApi.updateCourse(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['provider-courses'] }),
  })
}

export function useDeleteCourse() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => coursesApi.deleteCourse(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['provider-courses'] }),
  })
}

// -- Modules & Lessons --

export function useModules(courseId: string) {
  return useQuery({
    queryKey: ['modules', courseId],
    queryFn: () => coursesApi.getModules(courseId),
    enabled: !!courseId,
  })
}

export function useCreateModule(courseId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: coursesApi.CreateModuleDto) => coursesApi.createModule(courseId, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['modules', courseId] }),
  })
}

export function useUpdateModule(courseId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ moduleId, data }: { moduleId: string; data: coursesApi.UpdateModuleDto }) =>
      coursesApi.updateModule(courseId, moduleId, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['modules', courseId] }),
  })
}

export function useDeleteModule(courseId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (moduleId: string) => coursesApi.deleteModule(courseId, moduleId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['modules', courseId] }),
  })
}

export function useReorderModules(courseId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (order: coursesApi.ModuleOrder[]) => coursesApi.reorderModules(courseId, order),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['modules', courseId] }),
  })
}

export function useLessons(courseId: string, moduleId: string) {
  return useQuery({
    queryKey: ['lessons', courseId, moduleId],
    queryFn: () => coursesApi.getLessons(courseId, moduleId),
    enabled: !!courseId && !!moduleId,
  })
}

export function useCreateLesson(courseId: string, moduleId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: coursesApi.CreateLessonDto) =>
      coursesApi.createLesson(courseId, moduleId, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['lessons', courseId, moduleId] }),
  })
}

export function useDeleteLesson(courseId: string, moduleId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (lessonId: string) => coursesApi.deleteLesson(courseId, moduleId, lessonId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['lessons', courseId, moduleId] }),
  })
}

// -- Assignments --

export function useAssignments(courseId: string) {
  return useQuery({
    queryKey: ['provider-assignments', courseId],
    queryFn: () => gradingApi.listAssignments(courseId),
    enabled: !!courseId,
  })
}

export function useAssignment(id: string) {
  return useQuery({
    queryKey: ['provider-assignments', 'detail', id],
    queryFn: () => gradingApi.getAssignment(id),
    enabled: !!id,
  })
}

export function useCreateAssignment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: gradingApi.CreateAssignmentDto) => gradingApi.createAssignment(data),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: ['provider-assignments', vars.courseId] })
    },
  })
}

export function useUpdateAssignment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: gradingApi.UpdateAssignmentDto }) =>
      gradingApi.updateAssignment(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['provider-assignments'] }),
  })
}

// -- Submissions & Grading --

export function useSubmissions(assignmentId: string) {
  return useQuery({
    queryKey: ['submissions', assignmentId],
    queryFn: () => gradingApi.getSubmissions(assignmentId),
    enabled: !!assignmentId,
  })
}

export function usePendingSubmissions() {
  return useQuery({
    queryKey: ['pending-submissions'],
    queryFn: () => gradingApi.getAllPendingSubmissions(),
  })
}

export function useGradeSubmission() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      submissionId,
      data,
    }: {
      submissionId: string
      data: gradingApi.GradeSubmissionDto
    }) => gradingApi.gradeSubmission(submissionId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['submissions'] })
      qc.invalidateQueries({ queryKey: ['pending-submissions'] })
    },
  })
}

export function useGradeDistribution(courseId: string, assignmentId?: string) {
  return useQuery({
    queryKey: ['grade-distribution', courseId, assignmentId],
    queryFn: () => gradingApi.getDistribution(courseId, assignmentId),
    enabled: !!courseId,
  })
}

export function useGradingRules(courseId?: string) {
  return useQuery({
    queryKey: ['grading-rules', courseId],
    queryFn: () => gradingApi.getGradingRules(courseId),
  })
}

export function useCreateGradingRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: gradingApi.GradingRuleDto) => gradingApi.createGradingRule(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['grading-rules'] }),
  })
}

export function useUpdateGradingRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: gradingApi.GradingRuleDto }) =>
      gradingApi.updateGradingRule(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['grading-rules'] }),
  })
}

export function useDeleteGradingRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => gradingApi.deleteGradingRule(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['grading-rules'] }),
  })
}

// -- Video Sessions --

export function useVideoSessions(params?: { courseId?: string; status?: string }) {
  return useQuery({
    queryKey: ['video-sessions', params],
    queryFn: () => videoApi.listSessions(params),
  })
}

export function useVideoSession(id: string) {
  return useQuery({
    queryKey: ['video-sessions', id],
    queryFn: () => videoApi.getSession(id),
    enabled: !!id,
  })
}

export function useCreateVideoSession() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: videoApi.CreateSessionDto) => videoApi.createSession(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['video-sessions'] }),
  })
}

export function useCancelVideoSession() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => videoApi.cancelSession(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['video-sessions'] }),
  })
}

// -- Announcements --

export function useAnnouncements(courseId?: string) {
  return useQuery({
    queryKey: ['announcements', courseId],
    queryFn: () => announcementsApi.listAnnouncements(courseId),
  })
}

export function useCreateAnnouncement() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: announcementsApi.CreateAnnouncementDto) =>
      announcementsApi.createAnnouncement(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['announcements'] }),
  })
}

export function usePublishAnnouncement() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => announcementsApi.publishAnnouncement(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['announcements'] }),
  })
}

export function useDeleteAnnouncement() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => announcementsApi.deleteAnnouncement(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['announcements'] }),
  })
}

// -- Messages --

export function useInbox() {
  return useQuery({
    queryKey: ['messages', 'inbox'],
    queryFn: () => messagesApi.getInbox(),
  })
}

export function useSentThreads() {
  return useQuery({
    queryKey: ['messages', 'sent'],
    queryFn: () => messagesApi.getSentThreads(),
  })
}

export function useThread(threadId: string) {
  return useQuery({
    queryKey: ['messages', 'thread', threadId],
    queryFn: () => messagesApi.getThread(threadId),
    enabled: !!threadId,
  })
}

export function useCreateThread() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: messagesApi.CreateThreadDto) => messagesApi.createThread(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['messages', 'inbox'] })
      qc.invalidateQueries({ queryKey: ['messages', 'sent'] })
    },
  })
}

export function useSendMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ threadId, data }: { threadId: string; data: messagesApi.SendMessageDto }) =>
      messagesApi.sendMessage(threadId, data),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: ['messages', 'thread', vars.threadId] })
    },
  })
}

// -- Course analytics & roster --

export function useCourseAnalytics(courseId: string) {
  return useQuery({
    queryKey: ['course-analytics', courseId],
    queryFn: () => coursesApi.getCourseAnalytics(courseId),
    enabled: !!courseId,
  })
}

export function useCourseRoster(courseId: string) {
  return useQuery({
    queryKey: ['course-roster', courseId],
    queryFn: () => coursesApi.getCourseRoster(courseId),
    enabled: !!courseId,
  })
}

// -- Discussion threads --

export function useDiscussionThreads(courseId?: string) {
  return useQuery({
    queryKey: ['discussion-threads', courseId],
    queryFn: () => discussionApi.listThreads({ courseId, limit: 50 }),
  })
}

export function useCreateDiscussionThread() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: discussionApi.CreateThreadDto) => discussionApi.createThread(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['discussion-threads'] }),
  })
}

// -- Lecture Q&A --

export function useLectureQuestions(lectureId: string) {
  return useQuery({
    queryKey: ['lecture-questions', lectureId],
    queryFn: () => videoApi.listLectureQuestions(lectureId),
    enabled: !!lectureId,
    refetchInterval: 5000,
  })
}

export function useUpvoteQuestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ lectureId, questionId }: { lectureId: string; questionId: string }) =>
      videoApi.upvoteLectureQuestion(lectureId, questionId),
    onSuccess: (_, vars) =>
      qc.invalidateQueries({ queryKey: ['lecture-questions', vars.lectureId] }),
  })
}
