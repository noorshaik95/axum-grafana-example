import { apiClient } from './client'
import type { User, PaginatedResponse } from './types'

export async function listUsers(params?: {
  search?: string
  role?: string
  page?: number
  pageSize?: number
}): Promise<PaginatedResponse<User>> {
  const searchParams = new URLSearchParams()
  if (params?.search) searchParams.set('search', params.search)
  if (params?.role) searchParams.set('role', params.role)
  if (params?.page) searchParams.set('page', String(params.page))
  if (params?.pageSize) searchParams.set('pageSize', String(params.pageSize))
  const query = searchParams.toString()
  return apiClient.get<PaginatedResponse<User>>(`/api/users${query ? `?${query}` : ''}`)
}

export async function getUser(id: string): Promise<User> {
  return apiClient.get<User>(`/api/users/${id}`)
}

export async function updateUser(id: string, data: Partial<User>): Promise<User> {
  return apiClient.put<User>(`/api/users/${id}`, data)
}
