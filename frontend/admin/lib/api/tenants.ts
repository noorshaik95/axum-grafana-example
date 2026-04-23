import apiClient from './client'
import type {
  Tenant,
  TenantStatus,
  TenantUsage,
  TenantBillingHistory,
  ResourcePlan,
  PaginatedResponse,
  ListParams,
} from './types'

export interface ListTenantsParams extends ListParams {
  status?: TenantStatus
}

const NOT_IMPLEMENTED = 'Not available in MVP — backend endpoint not yet implemented'

export const tenantsApi = {
  async list(params?: ListTenantsParams): Promise<PaginatedResponse<Tenant>> {
    const response = await apiClient.get('/tenants', { params })
    return response.data
  },

  async get(id: string): Promise<Tenant> {
    const response = await apiClient.get(`/tenants/${id}`)
    return response.data
  },

  // No backend RPC for toggling tenant access yet (#52). Reject so the
  // mutation .catch() surfaces a red toast instead of a 404 in devtools.
  async toggleAccess(
    _id: string,
    _enabled: boolean,
    _reason: string
  ): Promise<{ success: boolean }> {
    return Promise.reject(new Error(NOT_IMPLEMENTED))
  },

  // #52 note: updatePlan stays network-bound because tenant-expert (#54)
  // is wiring the existing REST handler through the gateway. Once #54
  // lands the PUT succeeds; until then the user gets a 404/503 toast
  // from the apiClient interceptor. Do NOT toast-reject here.
  async updatePlan(id: string, plan: ResourcePlan): Promise<Tenant> {
    const response = await apiClient.put(`/tenants/${id}/plan`, plan)
    return response.data
  },

  async getUsage(id: string): Promise<TenantUsage> {
    const response = await apiClient.get(`/tenants/${id}/usage`)
    return response.data
  },

  async getBillingHistory(id: string): Promise<TenantBillingHistory> {
    const response = await apiClient.get(`/tenants/${id}/billing`)
    return response.data
  },

  async listUsers(
    id: string,
    params?: ListParams
  ): Promise<
    PaginatedResponse<{
      id: string
      email: string
      name: string
      role: string
      status: string
      lastLogin: string
    }>
  > {
    const response = await apiClient.get(`/tenants/${id}/users`, { params })
    return response.data
  },
}

export default tenantsApi
