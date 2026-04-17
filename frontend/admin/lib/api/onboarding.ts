import apiClient from './client'
import type {
  OnboardingJob,
  OnboardingPayload,
  OnboardingStatus,
  PaginatedResponse,
  ListParams,
} from './types'

export interface ListOnboardingParams extends ListParams {
  status?: OnboardingStatus
}

export const onboardingApi = {
  async list(params?: ListOnboardingParams): Promise<PaginatedResponse<OnboardingJob>> {
    const response = await apiClient.get('/onboarding', { params })
    return response.data
  },

  async get(id: string): Promise<OnboardingJob> {
    const response = await apiClient.get(`/onboarding/${id}`)
    return response.data
  },

  async start(payload: OnboardingPayload): Promise<OnboardingJob> {
    const response = await apiClient.post('/onboarding', payload)
    return response.data
  },

  async saveStep(id: string, step: number, data: Record<string, unknown>): Promise<OnboardingJob> {
    const response = await apiClient.put(`/onboarding/${id}/step/${step}`, data)
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
