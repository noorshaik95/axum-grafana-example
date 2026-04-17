import { get, patch } from './client';

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

export function getEnrolledCourses(): Promise<EnrolledCourse[]> {
  return get<EnrolledCourse[]>('/api/students/me/courses');
}

export function getCourseModules(courseId: string): Promise<CourseModule[]> {
  return get<CourseModule[]>(`/api/courses/${courseId}/modules`);
}

export function getModuleDetail(courseId: string, moduleId: string): Promise<CourseModule> {
  return get<CourseModule>(`/api/courses/${courseId}/modules/${moduleId}`);
}

export function markLessonComplete(courseId: string, lessonId: string): Promise<void> {
  return patch<void>(`/api/students/me/courses/${courseId}/progress`, {
    lessonId,
    completed: true,
  });
}

export function getProgress(): Promise<StudentProgress> {
  return get<StudentProgress>('/api/metrics/students/me');
}

export function getTimeOnTask(): Promise<TimeOnTask[]> {
  return get<TimeOnTask[]>('/api/metrics/students/me/time-on-task');
}
