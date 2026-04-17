import { apiClient } from './client'
import type {
  User,
  Role,
  AuthTokens,
  LoginRequest,
  RegisterRequest,
  UpdateProfileRequest,
} from './types'

// ---------------------------------------------------------------------------
// Raw API response types (snake_case from gateway)
// ---------------------------------------------------------------------------

interface RawUser {
  id: string
  email: string
  first_name?: string
  last_name?: string
  phone?: string
  is_active?: boolean
  auth_method?: string
  timezone?: string
  avatar_url?: string
  bio?: string
  organization_id?: string
  roles?:
    | string[]
    | Array<{ id: string; name: string; description?: string; permissions?: string[] }>
  created_at?: string
  updated_at?: string
}

interface RawAuthResponse {
  access_token?: string
  refresh_token?: string
  expires_in?: number
  // some endpoints may return camelCase already
  accessToken?: string
  refreshToken?: string
  expiresIn?: number
  user?: RawUser
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

function mapRoles(roles?: RawUser['roles']): readonly Role[] {
  if (!roles || roles.length === 0) return []
  const now = new Date().toISOString()
  return roles.map((r) => {
    if (typeof r === 'string') {
      return {
        id: '',
        name: r as Role['name'],
        description: null,
        permissions: [],
        createdAt: now,
        updatedAt: now,
      } as Role
    }
    return {
      id: r.id ?? '',
      name: r.name as Role['name'],
      description: r.description ?? null,
      permissions: r.permissions ?? [],
      createdAt: now,
      updatedAt: now,
    } as Role
  })
}

function mapUser(raw: RawUser): User {
  return {
    id: raw.id,
    email: raw.email,
    firstName: raw.first_name ?? '',
    lastName: raw.last_name ?? '',
    phone: raw.phone ?? null,
    isActive: raw.is_active ?? true,
    authMethod: (raw.auth_method as User['authMethod']) ?? 'normal',
    timezone: raw.timezone ?? 'UTC',
    avatarUrl: raw.avatar_url ?? null,
    bio: raw.bio ?? null,
    organizationId: raw.organization_id ?? null,
    roles: mapRoles(raw.roles),
    createdAt: raw.created_at ?? new Date().toISOString(),
    updatedAt: raw.updated_at ?? new Date().toISOString(),
  }
}

function mapAuthTokens(raw: RawAuthResponse): AuthTokens & { user: User } {
  const accessToken = raw.access_token ?? raw.accessToken ?? ''
  const refreshToken = raw.refresh_token ?? raw.refreshToken ?? ''
  const expiresIn = raw.expires_in ?? raw.expiresIn ?? 900
  const user = raw.user ? mapUser(raw.user) : ({} as User)
  return { accessToken, refreshToken, expiresIn, tokenType: 'Bearer', user }
}

// ---------------------------------------------------------------------------
// Auth API functions
// ---------------------------------------------------------------------------

export async function login(req: LoginRequest): Promise<AuthTokens & { user: User }> {
  const raw = await apiClient.post<RawAuthResponse>('/api/auth/login', req)
  const result = mapAuthTokens(raw)
  if (typeof window !== 'undefined' && result.accessToken) {
    localStorage.setItem('slate_token', result.accessToken)
    localStorage.setItem('slate_refresh_token', result.refreshToken)
  }
  return result
}

export async function register(req: RegisterRequest): Promise<AuthTokens & { user: User }> {
  const raw = await apiClient.post<RawAuthResponse>('/api/auth/register', req)
  const result = mapAuthTokens(raw)
  // Backend returns empty tokens after registration by design.
  // Only store tokens if they are non-empty (caller should auto-login after register).
  if (typeof window !== 'undefined' && result.accessToken) {
    localStorage.setItem('slate_token', result.accessToken)
    localStorage.setItem('slate_refresh_token', result.refreshToken)
  }
  return result
}

export async function refreshToken(token?: string): Promise<AuthTokens> {
  const refresh =
    token ?? (typeof window !== 'undefined' ? localStorage.getItem('slate_refresh_token') : null)
  const raw = await apiClient.post<RawAuthResponse>('/api/auth/refresh', {
    refreshToken: refresh,
  })
  const tokens = mapAuthTokens(raw)
  if (typeof window !== 'undefined' && tokens.accessToken) {
    localStorage.setItem('slate_token', tokens.accessToken)
    localStorage.setItem('slate_refresh_token', tokens.refreshToken)
  }
  return tokens
}

export async function logout(): Promise<void> {
  try {
    await apiClient.post<void>('/api/auth/logout')
  } finally {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('slate_token')
      localStorage.removeItem('slate_refresh_token')
    }
  }
}

export async function getProfile(): Promise<User> {
  const raw = await apiClient.get<RawUser>('/api/users/profile')
  return mapUser(raw)
}

export async function updateProfile(req: UpdateProfileRequest): Promise<User> {
  const raw = await apiClient.put<RawUser>('/api/users/profile', req)
  return mapUser(raw)
}
