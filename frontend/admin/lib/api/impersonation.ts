import apiClient from './client'
import type { ImpersonationSession, PaginatedResponse, ListParams } from './types'

export const impersonationApi = {
  async start(
    targetUserId: string,
    reason: string
  ): Promise<{ token: string; user: { id: string; name: string; tenantName: string } }> {
    const response = await apiClient.post('/impersonation/start', {
      targetUserId,
      reason,
    })
    return response.data
  },

  async end(): Promise<void> {
    await apiClient.post('/impersonation/end')
  },

  async listSessions(params?: ListParams): Promise<PaginatedResponse<ImpersonationSession>> {
    const response = await apiClient.get('/impersonation/sessions', { params })
    return response.data
  },
}

export default impersonationApi
