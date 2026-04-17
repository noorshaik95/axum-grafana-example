import apiClient from './client'
import type { AdminRole, AuditLogEntry, PaginatedResponse, ListParams } from './types'

export interface IAMUser {
  id: string
  email: string
  name: string
  role: string
  status: 'active' | 'inactive' | 'suspended'
  lastLogin?: string
  createdAt: string
}

export interface InviteUserRequest {
  email: string
  name: string
  role: string
}

export interface ListAuditParams extends ListParams {
  action?: string
  userId?: string
  startDate?: string
  endDate?: string
}

export const ADMIN_PERMISSIONS = [
  'manage_tenants',
  'manage_billing',
  'manage_users',
  'manage_roles',
  'view_metrics',
  'view_audit',
  'impersonate',
  'manage_system',
  'manage_onboarding',
  'manage_content',
] as const

export type AdminPermission = (typeof ADMIN_PERMISSIONS)[number]

export const iamApi = {
  // Users
  async listUsers(params?: ListParams): Promise<PaginatedResponse<IAMUser>> {
    const response = await apiClient.get('/iam/users', { params })
    return response.data
  },

  async inviteUser(data: InviteUserRequest): Promise<IAMUser> {
    const response = await apiClient.post('/iam/users/invite', data)
    return response.data
  },

  async updateUserStatus(userId: string, status: IAMUser['status']): Promise<void> {
    await apiClient.patch(`/iam/users/${userId}/status`, { status })
  },

  async deleteUser(id: string): Promise<void> {
    await apiClient.delete(`/iam/users/${id}`)
  },

  // Roles
  async listRoles(): Promise<AdminRole[]> {
    const response = await apiClient.get('/iam/roles')
    return response.data
  },

  async createRole(
    data: Pick<AdminRole, 'name' | 'description' | 'permissions'>
  ): Promise<AdminRole> {
    const response = await apiClient.post('/iam/roles', data)
    return response.data
  },

  async updateRolePermissions(roleId: string, permissions: string[]): Promise<AdminRole> {
    const response = await apiClient.patch(`/iam/roles/${roleId}/permissions`, {
      permissions,
    })
    return response.data
  },

  async deleteRole(id: string): Promise<void> {
    await apiClient.delete(`/iam/roles/${id}`)
  },

  // Audit
  async listAuditLogs(params?: ListAuditParams): Promise<PaginatedResponse<AuditLogEntry>> {
    const response = await apiClient.get('/iam/audit', { params })
    return response.data
  },
}

export default iamApi
