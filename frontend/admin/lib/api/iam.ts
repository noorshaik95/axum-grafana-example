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

const NOT_IMPLEMENTED = 'Admin write operations are not yet implemented'

export const iamApi = {
  // Users
  async listUsers(params?: ListParams): Promise<PaginatedResponse<IAMUser>> {
    const response = await apiClient.get('/admin/users', { params })
    return response.data
  },

  async inviteUser(_data: InviteUserRequest): Promise<IAMUser> {
    return Promise.reject(new Error(NOT_IMPLEMENTED))
  },

  async updateUserStatus(_userId: string, _status: IAMUser['status']): Promise<void> {
    return Promise.reject(new Error(NOT_IMPLEMENTED))
  },

  async deleteUser(_id: string): Promise<void> {
    return Promise.reject(new Error(NOT_IMPLEMENTED))
  },

  // Roles
  async listRoles(): Promise<AdminRole[]> {
    const response = await apiClient.get('/admin/roles')
    return response.data
  },

  async createRole(
    _data: Pick<AdminRole, 'name' | 'description' | 'permissions'>
  ): Promise<AdminRole> {
    return Promise.reject(new Error(NOT_IMPLEMENTED))
  },

  async updateRolePermissions(_roleId: string, _permissions: string[]): Promise<AdminRole> {
    return Promise.reject(new Error(NOT_IMPLEMENTED))
  },

  async deleteRole(_id: string): Promise<void> {
    return Promise.reject(new Error(NOT_IMPLEMENTED))
  },

  // Audit
  async listAuditLogs(params?: ListAuditParams): Promise<PaginatedResponse<AuditLogEntry>> {
    const response = await apiClient.get('/admin/audit', { params })
    return response.data
  },
}

export default iamApi
