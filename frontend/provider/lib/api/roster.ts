import { apiClient } from '../../../shared/lib/api/client'

// W12.1: metrics-service roster health.
// Response matches services/metrics-service/internal/service/roster_health.go.

export type RiskLevel = 'healthy' | 'slipping' | 'at_risk'

export interface RosterEntry {
  user_id: string
  display_name: string
  risk_level: RiskLevel
  missed_assignments: number
  days_since_active: number
  grade_trend: number[]
  suggested_action: string
  current_grade?: number | null
  last_active?: string | null
}

export interface RosterHealthResponse {
  course_id: string
  entries: RosterEntry[]
  at_risk: number
  slipping: number
  healthy: number
  updated_at: string
}

export interface ListRosterParams {
  course: string
  risk?: RiskLevel
  limit?: number
}

export async function getRosterHealth(params: ListRosterParams): Promise<RosterHealthResponse> {
  const q = new URLSearchParams()
  q.set('course', params.course)
  if (params.risk) q.set('risk', params.risk)
  if (params.limit) q.set('limit', String(params.limit))
  return apiClient.get<RosterHealthResponse>(`/api/metrics/roster-health?${q.toString()}`)
}

// T2 reject-with-toast: POST /api/roster/nudge has no gateway route today.
//   The metrics-service `SendNudge` HTTP handler exists service-local only;
//   flip back to an apiClient.post call once metrics-expert wires it through
//   the gateway (post-MVP).
export async function sendNudge(_userId: string, _courseId: string): Promise<void> {
  throw new Error('Not available in MVP — backend endpoint not yet implemented')
}
