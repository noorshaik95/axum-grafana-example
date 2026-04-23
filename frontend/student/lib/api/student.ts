import { get, patch, getCurrentUserId } from './client';

interface GatewayEnrollment {
  id: string;
  course_id: string;
  student_id: string;
  enrollment_type?: string;
  status?: string;
  enrolled_at?: string;
  section_id?: string;
}

interface GatewayCourseMetadata {
  course_code?: string;
  department?: string;
  max_students?: number;
  credits?: number;
  tags?: string[];
}

interface GatewayCourse {
  id: string;
  title: string;
  description?: string;
  term?: string;
  instructor_id?: string;
  metadata?: GatewayCourseMetadata;
}

interface GatewayEnrollmentWithCourse {
  enrollment: GatewayEnrollment;
  course: GatewayCourse;
}

interface GatewayStudentEnrollmentsResponse {
  enrollments: GatewayEnrollmentWithCourse[];
}

export interface EnrolledCourse {
  id: string;
  courseId: string;
  title: string;
  courseCode: string;
  instructorName: string;
  term: string;
  completionPercent: number;
  totalModules: number;
  completedModules: number;
  enrolledAt: string;
}

export interface CourseModule {
  id: string;
  courseId: string;
  name: string;
  description: string | null;
  displayOrder: number;
  lessons: ModuleLesson[];
}

export interface ModuleLesson {
  id: string;
  moduleId: string;
  name: string;
  description: string | null;
  displayOrder: number;
  resources: LessonResource[];
  completed: boolean;
}

export interface LessonResource {
  id: string;
  name: string;
  description: string | null;
  contentType: string;
  fileSize: number;
  manifestUrl: string | null;
  storageKey: string;
  durationSeconds: number | null;
  published: boolean;
}

export interface StudentProgress {
  totalCourses: number;
  completedCourses: number;
  overallCompletionPercent: number;
  gpaEstimate: number;
  streakDays: number;
  totalTimeMinutes: number;
  activityData: ActivityDay[];
}

export interface ActivityDay {
  date: string;
  count: number;
}

export interface TimeOnTask {
  courseId: string;
  courseTitle: string;
  totalMinutes: number;
}

export async function getEnrolledCourses(): Promise<EnrolledCourse[]> {
  const userId = getCurrentUserId();
  if (!userId) throw new Error('Not authenticated');
  const res = await get<GatewayStudentEnrollmentsResponse>(`/api/students/${userId}/enrollments`);
  const items = res?.enrollments ?? [];
  return items.map((item) => ({
    id: item.enrollment.id,
    courseId: item.course.id,
    title: item.course.title,
    courseCode: item.course.metadata?.course_code ?? '',
    instructorName: item.course.instructor_id ?? '',
    term: item.course.term ?? '',
    completionPercent: 0,
    totalModules: 0,
    completedModules: 0,
    enrolledAt: item.enrollment.enrolled_at ?? '',
  }));
}

export function getCourseModules(courseId: string): Promise<CourseModule[]> {
  return get<CourseModule[]>(`/api/courses/${courseId}/modules`);
}

export function getModuleDetail(courseId: string, moduleId: string): Promise<CourseModule> {
  return get<CourseModule>(`/api/courses/${courseId}/modules/${moduleId}`);
}

export function markLessonComplete(courseId: string, lessonId: string): Promise<void> {
  const userId = getCurrentUserId();
  if (!userId) throw new Error('Not authenticated');
  // TODO: no backend route for per-lesson completion; server-side progress
  // is aggregated via /api/metrics/students/:id/progress. Emit a no-op PATCH
  // against the user-scoped path so we can wire this up once backend lands.
  return patch<void>(`/api/students/${userId}/courses/${courseId}/progress`, {
    lessonId,
    completed: true,
  });
}

export function getProgress(): Promise<StudentProgress> {
  const userId = getCurrentUserId();
  if (!userId) throw new Error('Not authenticated');
  return get<StudentProgress>(`/api/metrics/students/${userId}/progress`);
}

export function getTimeOnTask(): Promise<TimeOnTask[]> {
  const userId = getCurrentUserId();
  if (!userId) throw new Error('Not authenticated');
  // TODO: no gateway route for time-on-task; leaving user-scoped path as a
  // placeholder until metrics-service exposes the RPC.
  return get<TimeOnTask[]>(`/api/metrics/students/${userId}/time-on-task`);
}

export interface NextLecture {
  sessionId: string;
  courseId: string;
  courseCode: string;
  title: string;
  instructorName: string;
  startsAt: string;
  joinUrl?: string;
}

export function getNextLecture(): Promise<NextLecture | null> {
  return get<NextLecture | null>('/api/courses/next-lecture');
}

export interface CourseNextUp {
  moduleId: string;
  lessonId: string;
  contentId?: string;
  title: string;
  resumePositionSeconds?: number;
}

export function getCourseNextUp(courseId: string, userId: string): Promise<CourseNextUp | null> {
  return get<CourseNextUp | null>(`/api/courses/${courseId}/next-up/${userId}`);
}
