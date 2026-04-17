import { get } from './client';

export interface CourseGradeSummary {
  courseId: string;
  courseTitle: string;
  courseCode: string;
  currentGrade: number;
  letterGrade: string;
  estimatedFinal: number;
  totalPoints: number;
  earnedPoints: number;
  completedAssignments: number;
  totalAssignments: number;
  breakdown: CategoryBreakdown[];
}

export interface CategoryBreakdown {
  category: string;
  weight: number;
  earned: number;
  total: number;
  percentage: number;
}

export interface GradeDistribution {
  ranges: DistributionRange[];
  classAverage: number;
  studentScore: number;
  studentPercentile: number;
  totalStudents: number;
}

export interface DistributionRange {
  label: string;
  min: number;
  max: number;
  count: number;
  isStudent: boolean;
}

export interface PerCourseGrade {
  assignmentId: string;
  assignmentTitle: string;
  maxPoints: number;
  score: number | null;
  adjustedScore: number | null;
  status: 'draft' | 'published' | 'ungraded';
  feedback: string | null;
  gradedAt: string | null;
  category: string;
  weight: number;
}

export function getGradesOverview(): Promise<CourseGradeSummary[]> {
  return get<CourseGradeSummary[]>('/api/students/me/grades');
}

export function getCourseGrades(courseId: string): Promise<PerCourseGrade[]> {
  return get<PerCourseGrade[]>(`/api/students/me/grades/${courseId}`);
}

export function getGradeDistribution(
  courseId: string,
  assignmentId?: string
): Promise<GradeDistribution> {
  const params = assignmentId ? `?assignmentId=${assignmentId}` : '';
  return get<GradeDistribution>(`/api/metrics/grades/distribution/${courseId}${params}`);
}

export function getGradeEstimate(courseId: string): Promise<{
  estimatedGrade: number;
  estimatedLetter: string;
  confidence: number;
}> {
  return get(`/api/students/me/grade-estimate/${courseId}`);
}
