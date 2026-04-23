'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as studentApi from './student';
import * as assignmentsApi from './assignments';
import * as gradesApi from './grades';
import * as videoApi from './video';
import * as announcementsApi from './announcements';
import * as messagesApi from './messages';
import * as progressApi from './progress';
import * as contentApi from './content';
import * as schedulingApi from './scheduling';

// Student / Courses
export function useEnrolledCourses() {
  return useQuery({
    queryKey: ['enrolled-courses'],
    queryFn: studentApi.getEnrolledCourses,
  });
}

export function useCourseModules(courseId: string) {
  return useQuery({
    queryKey: ['course-modules', courseId],
    queryFn: () => studentApi.getCourseModules(courseId),
    enabled: !!courseId,
  });
}

export function useModuleDetail(courseId: string, moduleId: string) {
  return useQuery({
    queryKey: ['module-detail', courseId, moduleId],
    queryFn: () => studentApi.getModuleDetail(courseId, moduleId),
    enabled: !!courseId && !!moduleId,
  });
}

export function useMarkLessonComplete() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { courseId: string; lessonId: string }) =>
      studentApi.markLessonComplete(vars.courseId, vars.lessonId),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: ['course-modules', vars.courseId] });
      queryClient.invalidateQueries({ queryKey: ['module-detail', vars.courseId] });
      queryClient.invalidateQueries({ queryKey: ['enrolled-courses'] });
      queryClient.invalidateQueries({ queryKey: ['student-progress'] });
    },
  });
}

export function useStudentProgress() {
  return useQuery({
    queryKey: ['student-progress'],
    queryFn: studentApi.getProgress,
  });
}

// Assignments
export function useStudentAssignments(params?: {
  courseId?: string;
  status?: string;
  upcoming?: boolean;
}) {
  return useQuery({
    queryKey: ['student-assignments', params],
    queryFn: () => assignmentsApi.getAssignments(params),
  });
}

export function useStudentAssignment(id: string) {
  return useQuery({
    queryKey: ['student-assignment', id],
    queryFn: () => assignmentsApi.getAssignment(id),
    enabled: !!id,
  });
}

export function useAssignmentSubmissions(assignmentId: string) {
  return useQuery({
    queryKey: ['assignment-submissions', assignmentId],
    queryFn: () => assignmentsApi.getSubmissions(assignmentId),
    enabled: !!assignmentId,
  });
}

export function useSubmitAssignment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { assignmentId: string; files: File[] }) =>
      assignmentsApi.submitAssignment(vars.assignmentId, vars.files),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({
        queryKey: ['assignment-submissions', vars.assignmentId],
      });
      queryClient.invalidateQueries({ queryKey: ['student-assignments'] });
      queryClient.invalidateQueries({ queryKey: ['student-assignment', vars.assignmentId] });
    },
  });
}

export function useGradedResult(assignmentId: string, submissionId: string) {
  return useQuery({
    queryKey: ['graded-result', assignmentId, submissionId],
    queryFn: () => assignmentsApi.getGradedResult(assignmentId, submissionId),
    enabled: !!assignmentId && !!submissionId,
  });
}

// Grades
export function useGradesOverview() {
  return useQuery({
    queryKey: ['grades-overview'],
    queryFn: gradesApi.getGradesOverview,
  });
}

export function useCourseGrades(courseId: string) {
  return useQuery({
    queryKey: ['course-grades', courseId],
    queryFn: () => gradesApi.getCourseGrades(courseId),
    enabled: !!courseId,
  });
}

export function useGradeDistribution(courseId: string, assignmentId?: string) {
  return useQuery({
    queryKey: ['grade-distribution', courseId, assignmentId],
    queryFn: () => gradesApi.getGradeDistribution(courseId, assignmentId),
    enabled: !!courseId,
  });
}

// Video
export function useVideoSessions(params?: { upcoming?: boolean; courseId?: string }) {
  return useQuery({
    queryKey: ['video-sessions', params],
    queryFn: () => videoApi.getSessions(params),
  });
}

export function useVideoSession(sessionId: string) {
  return useQuery({
    queryKey: ['video-session', sessionId],
    queryFn: () => videoApi.getSession(sessionId),
    enabled: !!sessionId,
  });
}

// Announcements
export function useAnnouncements(params?: { courseId?: string; unreadOnly?: boolean }) {
  return useQuery({
    queryKey: ['announcements', params],
    queryFn: () => announcementsApi.getAnnouncements(params),
  });
}

// Messages
export function useInbox() {
  return useQuery({
    queryKey: ['messages', 'inbox'],
    queryFn: messagesApi.getInbox,
  });
}

export function useSentMessages() {
  return useQuery({
    queryKey: ['messages', 'sent'],
    queryFn: messagesApi.getSent,
  });
}

export function useMessageThread(threadId: string) {
  return useQuery({
    queryKey: ['messages', 'thread', threadId],
    queryFn: () => messagesApi.getThread(threadId),
    enabled: !!threadId,
  });
}

export function useSendMessage() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: messagesApi.sendMessage,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['messages'] });
    },
  });
}

export function useReplyToThread() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { threadId: string; content: string }) =>
      messagesApi.replyToThread(vars.threadId, vars.content),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: ['messages', 'thread', vars.threadId] });
      queryClient.invalidateQueries({ queryKey: ['messages', 'inbox'] });
    },
  });
}

// Progress
export function useCompletionStats() {
  return useQuery({
    queryKey: ['completion-stats'],
    queryFn: progressApi.getCompletionStats,
  });
}

export function useActivityCalendar(days?: number) {
  return useQuery({
    queryKey: ['activity-calendar', days],
    queryFn: () => progressApi.getActivityCalendar(days),
  });
}

export function useTimeOnTask() {
  return useQuery({
    queryKey: ['time-on-task'],
    queryFn: progressApi.getTimeOnTask,
  });
}

// Content
export function useContent(contentId: string) {
  return useQuery({
    queryKey: ['content', contentId],
    queryFn: () => contentApi.getContent(contentId),
    enabled: !!contentId,
  });
}

export function useUpdatePosition() {
  return useMutation({
    mutationFn: (vars: { contentId: string; positionSeconds: number }) =>
      contentApi.putPosition(vars.contentId, vars.positionSeconds),
  });
}

// Video / live lecture
export function useNextLecture() {
  return useQuery({
    queryKey: ['next-lecture'],
    queryFn: studentApi.getNextLecture,
  });
}

// Scheduling / office hours
export function useOfficeHoursSlots(params?: {
  instructorId?: string;
  courseId?: string;
  dateFrom?: string;
  dateTo?: string;
}) {
  return useQuery({
    queryKey: ['office-hours-slots', params],
    queryFn: () => schedulingApi.getSlots(params),
  });
}
