import { get, getCurrentUserId } from './client';
import { getEnrolledCourses } from './student';

interface GatewayGradebookEntry {
  assignment_id: string;
  assignment_title: string;
  max_points: number;
  score?: number;
  adjusted_score?: number;
  status?: string;
  due_date?: string;
  submitted_at?: string;
  is_late?: boolean;
}

interface GatewayStudentGradebookResponse {
  student_id: string;
  course_id: string;
  entries: GatewayGradebookEntry[];
  total_points: number;
  earned_points: number;
  percentage: number;
  letter_grade: string;
}

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

export async function getGradesOverview(): Promise<CourseGradeSummary[]> {
  const userId = getCurrentUserId();
  if (!userId) throw new Error('Not authenticated');
  const courses = await getEnrolledCourses();
  const results = await Promise.all(
    courses.map(async (course) => {
      const res = await get<GatewayStudentGradebookResponse>(
        `/api/students/${userId}/gradebook?course_id=${encodeURIComponent(course.courseId)}`
      );
      const entries = res?.entries ?? [];
      const completed = entries.filter(
        (e) => e.status && e.status !== 'not_graded' && e.status !== 'ungraded'
      ).length;
      const earned = res?.earned_points ?? 0;
      const total = res?.total_points ?? 0;
      const currentGrade = res?.percentage ?? 0;
      const summary: CourseGradeSummary = {
        courseId: course.courseId,
        courseTitle: course.title,
        courseCode: course.courseCode,
        currentGrade,
        letterGrade: res?.letter_grade ?? '',
        estimatedFinal: currentGrade,
        totalPoints: total,
        earnedPoints: earned,
        completedAssignments: completed,
        totalAssignments: entries.length,
        breakdown: [],
      };
      return summary;
    })
  );
  return results;
}

export async function getCourseGrades(courseId: string): Promise<PerCourseGrade[]> {
  const userId = getCurrentUserId();
  if (!userId) throw new Error('Not authenticated');
  const res = await get<GatewayStudentGradebookResponse>(
    `/api/students/${userId}/gradebook?course_id=${encodeURIComponent(courseId)}`
  );
  const entries = res?.entries ?? [];
  return entries.map((e) => {
    const rawStatus = e.status ?? 'ungraded';
    const status: PerCourseGrade['status'] =
      rawStatus === 'published' ? 'published' : rawStatus === 'draft' ? 'draft' : 'ungraded';
    return {
      assignmentId: e.assignment_id,
      assignmentTitle: e.assignment_title,
      maxPoints: e.max_points,
      score: e.score ?? null,
      adjustedScore: e.adjusted_score ?? null,
      status,
      feedback: null,
      gradedAt: null,
      category: '',
      weight: 0,
    };
  });
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
  const userId = getCurrentUserId();
  if (!userId) throw new Error('Not authenticated');
  // TODO: no gateway route for per-student grade estimate; metrics-service
  // only exposes course-level distributions. Leaving user-scoped path as a
  // placeholder until the backend RPC lands.
  return get(`/api/students/${userId}/grade-estimate/${courseId}`);
}
