import { get } from './client';

export interface CompletionStats {
  totalCourses: number;
  completedCourses: number;
  totalLessons: number;
  completedLessons: number;
  overallPercent: number;
  gpaEstimate: number;
  streakDays: number;
}

export interface ActivityDay {
  date: string;
  count: number;
}

export interface TimeOnTaskEntry {
  courseId: string;
  courseTitle: string;
  totalMinutes: number;
}

export function getCompletionStats(): Promise<CompletionStats> {
  return get<CompletionStats>('/api/metrics/students/me/completion');
}

export function getActivityCalendar(days?: number): Promise<ActivityDay[]> {
  const params = days ? `?days=${days}` : '';
  return get<ActivityDay[]>(`/api/metrics/students/me/activity${params}`);
}

export function getTimeOnTask(): Promise<TimeOnTaskEntry[]> {
  return get<TimeOnTaskEntry[]>('/api/metrics/students/me/time-on-task');
}
