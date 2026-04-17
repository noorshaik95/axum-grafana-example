import { get, post } from './client';

export interface StudentAssignment {
  id: string;
  courseId: string;
  courseTitle: string;
  title: string;
  description: string | null;
  maxPoints: number;
  dueDate: string;
  latePenaltyPercent: number;
  maxLateDays: number;
  status: 'not_started' | 'in_progress' | 'submitted' | 'graded';
  submissionCount: number;
  grade: number | null;
  feedback: string | null;
}

export interface SubmissionRecord {
  id: string;
  assignmentId: string;
  filePath: string;
  fileName: string;
  submittedAt: string;
  status: 'submitted' | 'graded' | 'returned';
  isLate: boolean;
  daysLate: number;
}

export interface GradedResult {
  score: number;
  adjustedScore: number;
  maxPoints: number;
  feedback: string | null;
  rubricBreakdown: RubricItem[] | null;
  gradedAt: string;
  gradedBy: string;
}

export interface RubricItem {
  criterion: string;
  maxPoints: number;
  earnedPoints: number;
  comment: string | null;
}

export function getAssignments(params?: {
  courseId?: string;
  status?: string;
  upcoming?: boolean;
}): Promise<StudentAssignment[]> {
  const searchParams = new URLSearchParams();
  if (params?.courseId) searchParams.set('courseId', params.courseId);
  if (params?.status) searchParams.set('status', params.status);
  if (params?.upcoming) searchParams.set('upcoming', 'true');
  const query = searchParams.toString();
  return get<StudentAssignment[]>(`/api/assignments${query ? `?${query}` : ''}`);
}

export function getAssignment(id: string): Promise<StudentAssignment> {
  return get<StudentAssignment>(`/api/assignments/${id}`);
}

export function getSubmissions(assignmentId: string): Promise<SubmissionRecord[]> {
  return get<SubmissionRecord[]>(`/api/assignments/${assignmentId}/submissions`);
}

export function submitAssignment(assignmentId: string, files: File[]): Promise<SubmissionRecord> {
  const form = new FormData();
  files.forEach((f) => form.append('files', f));
  return post<SubmissionRecord>(`/api/assignments/${assignmentId}/submissions`, form);
}

export function getGradedResult(assignmentId: string, submissionId: string): Promise<GradedResult> {
  return get<GradedResult>(`/api/assignments/${assignmentId}/submissions/${submissionId}/grade`);
}
