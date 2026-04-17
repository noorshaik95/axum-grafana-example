import { apiClient } from '../../../../shared/lib/api/client'

export { apiClient, ApiError } from '../../../../shared/lib/api/client'

export function getTenantId(): string | null {
  if (typeof window === 'undefined') return null
  const token = localStorage.getItem('slate_token')
  if (!token) return null
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.tenantId ?? payload.tenant_id ?? null
  } catch {
    return null
  }
}
