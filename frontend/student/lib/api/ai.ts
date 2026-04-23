import { get, post } from './client';

export interface WelcomeMessage {
  message: string;
  cta?: { label: string; href: string } | null;
}

export interface GradeProjection {
  currentGrade: number;
  projectedGrade: number;
  projectedLetter: string;
  requiredScores: Array<{
    assignmentId: string;
    assignmentTitle: string;
    maxPoints: number;
    requiredScore: number;
    requiredPct: number;
  }>;
  confidence: number;
  rationale: string;
}

export interface StudyPlanRequest {
  courseIds?: string[];
  goalLetter?: string;
  hoursPerWeek?: number;
}

export interface StudyPlanWeek {
  num: number;
  theme: string;
  status: 'done' | 'current' | 'upcoming';
  tasks: Array<{ id: string; title: string; courseCode?: string; dueLabel?: string }>;
}

export interface StudyPlan {
  weeks: StudyPlanWeek[];
  generatedAt: string;
}

export interface CmdPaletteResult {
  id: string;
  label: string;
  description?: string;
  route?: string;
  group?: string;
}

export interface CmdPaletteRequest {
  query: string;
  userContext?: { role?: string; courseIds?: string[] };
  routeCatalog: Array<{ path: string; label: string; group?: string }>;
}

export function getWelcome(): Promise<WelcomeMessage> {
  return get<WelcomeMessage>('/api/ai/welcome');
}

export function getGradeProjection(courseId: string): Promise<GradeProjection> {
  const params = new URLSearchParams({ course: courseId });
  return get<GradeProjection>(`/api/ai/grade-projection?${params.toString()}`);
}

export function generateStudyPlan(body: StudyPlanRequest): Promise<StudyPlan> {
  return post<StudyPlan>('/api/ai/study-plan', body);
}

export function queryCmdPalette(body: CmdPaletteRequest): Promise<{ results: CmdPaletteResult[] }> {
  return post<{ results: CmdPaletteResult[] }>('/api/ai/cmd-palette', body);
}
