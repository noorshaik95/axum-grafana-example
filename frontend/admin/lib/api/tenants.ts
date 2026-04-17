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

export const tenantsApi = {
  async list(params?: ListTenantsParams): Promise<PaginatedResponse<Tenant>> {
    const response = await apiClient.get('/tenants', { params })
    return response.data
  },

  async get(id: string): Promise<Tenant> {
    const response = await apiClient.get(`/tenants/${id}`)
    return response.data
  },

  async toggleAccess(id: string, enabled: boolean, reason: string): Promise<{ success: boolean }> {
    const response = await apiClient.patch(`/tenants/${id}/access`, {
      enabled,
      reason,
    })
    return response.data
  },

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
