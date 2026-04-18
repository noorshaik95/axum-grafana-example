import { apiClient } from '../../../shared/lib/api/client'
import type { Assignment, Submission, Grade } from '../../../shared/lib/api/types'

export interface CreateAssignmentDto {
  courseId: string
  title: string
  description?: string
  maxPoints: number
  dueDate: string
  latePenaltyPercent?: number
  maxLateDays?: number
  rubric?: RubricCriterion[]
}

export interface UpdateAssignmentDto {
  title?: string
  description?: string
  maxPoints?: number
  dueDate?: string
  latePenaltyPercent?: number
  maxLateDays?: number
  rubric?: RubricCriterion[]
}

export interface RubricCriterion {
  name: string
  description: string
  maxPoints: number
}

export interface GradeSubmissionDto {
  score: number
  feedback?: string
  rubricScores?: Record<string, number>
  override?: boolean
  overrideJustification?: string
}

export interface GradingRule {
  id: string
  courseId: string | null
  assignmentType: string
  weight: number
  createdAt: string
  updatedAt: string
}

export interface GradeScale {
  letter: string
  minScore: number
}

export interface GradingRuleDto {
  courseId?: string
  assignmentType: string
  weight: number
}

export interface GradeDistribution {
  buckets: { range: string; count: number }[]
  mean: number
  median: number
  p25: number
  p75: number
  stdDev: number
}

export async function listAssignments(courseId: string): Promise<Assignment[]> {
  return apiClient.get<Assignment[]>(`/api/assignments?courseId=${courseId}`)
}

export async function getAssignment(id: string): Promise<Assignment> {
  return apiClient.get<Assignment>(`/api/assignments/${id}`)
}

export async function createAssignment(data: CreateAssignmentDto): Promise<Assignment> {
  return apiClient.post<Assignment>('/api/assignments', data)
}

export async function updateAssignment(id: string, data: UpdateAssignmentDto): Promise<Assignment> {
  return apiClient.put<Assignment>(`/api/assignments/${id}`, data)
}

export async function deleteAssignment(id: string): Promise<void> {
  return apiClient.delete(`/api/assignments/${id}`)
}

export async function getSubmissions(assignmentId: string): Promise<Submission[]> {
  return apiClient.get<Submission[]>(`/api/assignments/${assignmentId}/submissions`)
}

export async function getAllPendingSubmissions(): Promise<Submission[]> {
  return apiClient.get<Submission[]>('/api/submissions?status=submitted')
}

export async function gradeSubmission(
  submissionId: string,
  data: GradeSubmissionDto
): Promise<Grade> {
  return apiClient.post<Grade>(`/api/submissions/${submissionId}/grade`, data)
}

export async function autoGrade(assignmentId: string): Promise<void> {
  return apiClient.post(`/api/assignments/${assignmentId}/auto-grade`)
}

export async function getDistribution(
  courseId: string,
  assignmentId?: string
): Promise<GradeDistribution> {
  const params = assignmentId ? `?assignmentId=${assignmentId}` : ''
  return apiClient.get<GradeDistribution>(`/api/metrics/grades/distribution/${courseId}${params}`)
}

export async function getGradingRules(courseId?: string): Promise<GradingRule[]> {
  const params = courseId ? `?courseId=${courseId}` : ''
  return apiClient.get<GradingRule[]>(`/api/grading-rules${params}`)
}

export async function createGradingRule(data: GradingRuleDto): Promise<GradingRule> {
  return apiClient.post<GradingRule>('/api/grading-rules', data)
}

export async function updateGradingRule(id: string, data: GradingRuleDto): Promise<GradingRule> {
  return apiClient.put<GradingRule>(`/api/grading-rules/${id}`, data)
}

export async function deleteGradingRule(id: string): Promise<void> {
  return apiClient.delete(`/api/grading-rules/${id}`)
}
