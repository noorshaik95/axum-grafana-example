export { apiClient, ApiError } from '../../../shared/lib/api/client'

export * as gradingApi from './grading'
export * as rosterApi from './roster'
export * as schedulingApi from './scheduling'
export * as teachApi from './teach'

// T3-R1: decode claims from the access token so the provider shell can render
// without blocking on GET /api/users/profile (which is 404 until the user-auth
// team fixes the seed/route mismatch — tracked in task #57). Claim shape matches
// services/user-auth-service/pkg/jwt/jwt.go::Claims.
export interface JwtClaims {
  userId: string | null
  email: string | null
  roles: string[]
  tenantId: string | null
  firstName: string | null
  lastName: string | null
}

const EMPTY_CLAIMS: JwtClaims = {
  userId: null,
  email: null,
  roles: [],
  tenantId: null,
  firstName: null,
  lastName: null,
}

function base64UrlDecode(segment: string): string | null {
  try {
    let payload = segment.replace(/-/g, '+').replace(/_/g, '/')
    const pad = payload.length % 4
    if (pad) payload += '='.repeat(4 - pad)
    return atob(payload)
  } catch {
    return null
  }
}

export function decodeJwtClaims(token: string | null | undefined): JwtClaims {
  if (!token) return EMPTY_CLAIMS
  const parts = token.split('.')
  if (parts.length < 2) return EMPTY_CLAIMS
  const raw = base64UrlDecode(parts[1])
  if (!raw) return EMPTY_CLAIMS
  try {
    const p = JSON.parse(raw) as Record<string, unknown>
    const roles = Array.isArray(p.roles)
      ? ((p.roles as unknown[]).filter((r) => typeof r === 'string') as string[])
      : []
    const email = typeof p.email === 'string' ? p.email : null
    const [localPart] = email ? email.split('@') : [null]
    const [firstGuess, lastGuess] = localPart ? localPart.split('.') : [null, null]
    return {
      userId: typeof p.user_id === 'string' ? p.user_id : typeof p.sub === 'string' ? p.sub : null,
      email,
      roles,
      tenantId:
        typeof p.tenant_id === 'string'
          ? p.tenant_id
          : typeof p.tenantId === 'string'
            ? (p.tenantId as string)
            : null,
      firstName: typeof p.first_name === 'string' ? p.first_name : (firstGuess ?? null),
      lastName: typeof p.last_name === 'string' ? p.last_name : (lastGuess ?? null),
    }
  } catch {
    return EMPTY_CLAIMS
  }
}

export function getStoredToken(): string | null {
  if (typeof window === 'undefined') return null
  return localStorage.getItem('slate_token')
}

export function getCurrentClaims(): JwtClaims {
  return decodeJwtClaims(getStoredToken())
}

export function hasInstructorRole(claims: JwtClaims): boolean {
  return claims.roles.some((r) => r === 'instructor' || r === 'admin' || r === 'superadmin')
}

export function getTenantId(): string | null {
  return getCurrentClaims().tenantId
}
