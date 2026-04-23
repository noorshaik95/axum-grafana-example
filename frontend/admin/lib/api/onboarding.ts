import apiClient from './client'
import type { OnboardingJob, OnboardingStatus, PaginatedResponse, ListParams } from './types'

export interface ListOnboardingParams extends ListParams {
  status?: OnboardingStatus
}

export interface StartOnboardingRequest {
  institution_name: string
  admin_email: string
}

export interface StartOnboardingResponse {
  id: string
  job?: OnboardingJob
  raw?: Record<string, unknown>
}

/**
 * Onboarding API — backed by Restate Virtual Objects (#47).
 *
 * The gateway exposes per-workflow Restate routes keyed by a client-generated
 * UUID (plan/onboarding-service.md:173):
 *   POST /api/onboarding/:id/start        — create workflow
 *   POST /api/onboarding/:id/save-step    — persist step data
 *   POST /api/onboarding/:id/submit       — submit for review
 *   POST /api/onboarding/:id/approve      — approve
 *   POST /api/onboarding/:id/reject       — reject with reason
 *   POST /api/onboarding/:id/complete-sso — finish SSO config
 *   POST /api/onboarding/:id/reopen       — reopen a rejected workflow
 *   GET  /api/onboarding/:id/status       — read current state
 *
 * There is NO collection/projection endpoint (no `GET /api/onboarding`,
 * no `GET /api/onboarding/:id`) — those are the responsibility of a
 * separate projection service that is not yet shipped. `list` and `get`
 * below degrade gracefully: `list` returns an empty paginated response so
 * the UI renders the "no jobs yet" empty state instead of crashing on a
 * 404; `get` delegates to `getStatus` which IS a real route.
 *
 * TODO(#47): flip `list`/`get` to real routes once the projection backend
 * ships (Option B in task brief — tenant-expert / onboarding-expert).
 */
function newWorkflowId(): string {
  const c: any = (globalThis as any).crypto
  if (c && typeof c.randomUUID === 'function') return c.randomUUID()
  // Fallback for older browsers / SSR — matches UUID v4 shape.
  const rand = (n: number) => {
    const arr = new Uint8Array(n)
    if (c && typeof c.getRandomValues === 'function') {
      c.getRandomValues(arr)
    } else {
      for (let i = 0; i < n; i += 1) arr[i] = Math.floor(Math.random() * 256)
    }
    return arr
  }
  const b = rand(16)
  b[6] = (b[6] & 0x0f) | 0x40
  b[8] = (b[8] & 0x3f) | 0x80
  const hex = Array.from(b).map((x) => x.toString(16).padStart(2, '0'))
  return `${hex.slice(0, 4).join('')}-${hex.slice(4, 6).join('')}-${hex.slice(6, 8).join('')}-${hex.slice(8, 10).join('')}-${hex.slice(10, 16).join('')}`
}

export const onboardingApi = {
  async list(_params?: ListOnboardingParams): Promise<PaginatedResponse<OnboardingJob>> {
    // No projection endpoint yet — return empty rather than 404. UI should
    // render the empty state + inform the user. See TODO(#47) above.
    return {
      data: [],
      total: 0,
      page: 1,
      pageSize: 0,
      totalPages: 0,
    }
  },

  async get(id: string): Promise<OnboardingJob> {
    // No direct projection; surface whatever the status endpoint knows.
    // Callers using rich OnboardingJob fields should prefer `getStatus`.
    const status = await onboardingApi.getStatus(id)
    return status.job
  },

  async start(body: StartOnboardingRequest): Promise<StartOnboardingResponse> {
    // Restate OnboardingWorkflow::start expects a minimal snake_case payload
    // matching services/onboarding-service/src/state.rs::StartPayload:
    //   { institution_name: String, admin_email: String }
    // Callers (wizards) are responsible for passing those exact keys + for
    // persisting richer wizard data via subsequent `saveStep(id, 1, ...)`
    // calls so the workflow has full context available downstream.
    const id = newWorkflowId()
    const response = await apiClient.post(`/onboarding/${id}/start`, body)
    const resData = (response.data ?? {}) as Record<string, unknown>
    const maybeJob = (resData.job ?? resData) as OnboardingJob | undefined
    return { id, job: maybeJob, raw: resData }
  },

  async saveStep(id: string, step: number, data: Record<string, unknown>): Promise<OnboardingJob> {
    const response = await apiClient.post(`/onboarding/${id}/save-step`, {
      step,
      data,
    })
    return response.data
  },

  async submit(id: string): Promise<OnboardingJob> {
    const response = await apiClient.post(`/onboarding/${id}/submit`)
    return response.data
  },

  async approve(id: string): Promise<OnboardingJob> {
    const response = await apiClient.post(`/onboarding/${id}/approve`)
    return response.data
  },

  async reject(id: string, reason: string): Promise<OnboardingJob> {
    const response = await apiClient.post(`/onboarding/${id}/reject`, { reason })
    return response.data
  },

  async completeSso(id: string, data: Record<string, unknown>): Promise<OnboardingJob> {
    const response = await apiClient.post(`/onboarding/${id}/complete-sso`, data)
    return response.data
  },

  async reopen(id: string): Promise<OnboardingJob> {
    const response = await apiClient.post(`/onboarding/${id}/reopen`)
    return response.data
  },

  async getStatus(id: string): Promise<{
    job: OnboardingJob
    steps: Array<{
      step: number
      title: string
      status: 'completed' | 'in_progress' | 'pending' | 'failed'
      data?: Record<string, unknown>
    }>
  }> {
    const response = await apiClient.get(`/onboarding/${id}/status`)
    return response.data
  },
}

export default onboardingApi
