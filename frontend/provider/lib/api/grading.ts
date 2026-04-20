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

// --- W9: grading queue + batch ---

export interface QueuePatternSubmission {
  submission_id: string
  student_id: string
  student_name?: string
  score?: number
  is_late?: boolean
}

export interface QueuePatternGroup {
  pattern_id: string
  description: string
  count: number
  auto_score_suggestion: number
  submissions: QueuePatternSubmission[]
}

export interface QueueCountResponse {
  assignment_id: string
  pending: number
  patterns: number
}

export interface QueueResponse {
  assignment_id: string
  patterns: QueuePatternGroup[]
  total_pending: number
}

export interface BatchRubricScore {
  row_id: string
  points: number
}

export interface BatchGradeRequest {
  pattern_id: string
  assignment_id: string
  submission_ids?: string[]
  rubric_scores: BatchRubricScore[]
  feedback_template: string
  graded_by?: string
  instructor_id?: string
}

export interface BatchGradeResult {
  pattern_id: string
  assignment_id: string
  graded_count: number
  grade_ids: string[]
}

export interface GradeDraft {
  submission_id: string
  rubric_scores: BatchRubricScore[]
  feedback: string
  updated_at: string
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

// --- W9.4: pattern-grouped queue ---

// NOTE: `/api/grading/queue/count` is defined in gateway-config.yaml but the
// underlying gRPC method doesn't exist yet (gateway has no HTTP-passthrough;
// tracked in CARRYOVER §1a + task #29 proto audit). Callers must supply an
// assignmentId today; the /teach aggregate counts assignments client-side until
// gateway-proxy-expert lands option A. Passing an empty assignmentId throws
// client-side rather than 404ing on the wire.
export async function getQueueCount(params: {
  assignmentId: string
  instructorId?: string
}): Promise<QueueCountResponse> {
  if (!params.assignmentId) {
    throw new Error(
      'getQueueCount requires assignmentId until /api/grading/queue/count aggregate is wired'
    )
  }
  const q = new URLSearchParams()
  q.set('assignment_id', params.assignmentId)
  if (params.instructorId) q.set('instructor_id', params.instructorId)
  return apiClient.get<QueueCountResponse>(`/api/grading/queue/count?${q.toString()}`)
}

export async function getQueue(
  assignmentId: string,
  instructorId?: string
): Promise<QueueResponse> {
  const q = new URLSearchParams()
  if (instructorId) q.set('instructor_id', instructorId)
  const suffix = q.toString() ? `?${q.toString()}` : ''
  return apiClient.get<QueueResponse>(`/api/grading/queue/${assignmentId}${suffix}`)
}

// --- W9.5: batch grade apply ---

export async function applyBatch(data: BatchGradeRequest): Promise<BatchGradeResult> {
  return apiClient.post<BatchGradeResult>('/api/grades/batch', data)
}

// TODO(wave-4): the assignment-grading service has not shipped a drafts endpoint
//   yet. Once W9.6 lands, `listDrafts` + `saveDraft` will call
//   `GET /api/grades/drafts?assignmentId=...` and `PUT /api/grades/drafts/:subId`
//   respectively. For now they resolve to empty / no-op so the UI doesn't break.
export async function listDrafts(_assignmentId: string): Promise<GradeDraft[]> {
  return []
}

export async function saveDraft(_submissionId: string, _draft: Partial<GradeDraft>): Promise<void> {
  return
}
