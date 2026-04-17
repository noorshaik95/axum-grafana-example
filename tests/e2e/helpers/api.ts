/**
 * Direct API helper for test setup and teardown.
 * Makes fetch calls to the API gateway for operations like user registration
 * that should bypass the UI.
 */

const API_URL = process.env.API_URL ?? 'http://localhost:8080'

export interface AuthTokens {
  access_token: string
  refresh_token: string
}

export interface RegisterPayload {
  email: string
  password: string
  first_name: string
  last_name: string
  phone?: string
}

export interface LoginPayload {
  email: string
  password: string
}

/**
 * Register a new user via the API. Returns tokens on success.
 * Swallows "already exists" errors gracefully.
 */
export async function registerUser(payload: RegisterPayload): Promise<AuthTokens | null> {
  const res = await fetch(`${API_URL}/api/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })

  if (!res.ok) {
    const body = await res.text()
    // Tolerate duplicate registrations — the user already exists from a prior run
    if (res.status === 409 || body.includes('already exists') || body.includes('duplicate')) {
      return null
    }
    throw new Error(`Registration failed (${res.status}): ${body}`)
  }

  return res.json()
}

/**
 * Login an existing user. Returns tokens.
 */
export async function loginUser(payload: LoginPayload): Promise<AuthTokens> {
  const res = await fetch(`${API_URL}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })

  if (!res.ok) {
    const body = await res.text()
    throw new Error(`Login failed (${res.status}): ${body}`)
  }

  return res.json()
}

/**
 * Refresh an access token.
 */
export async function refreshToken(refresh_token: string): Promise<AuthTokens> {
  const res = await fetch(`${API_URL}/api/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token }),
  })

  if (!res.ok) {
    const body = await res.text()
    throw new Error(`Token refresh failed (${res.status}): ${body}`)
  }

  return res.json()
}

/**
 * Fetch the user profile for a given token.
 */
export async function getUserProfile(token: string): Promise<Record<string, unknown>> {
  const res = await fetch(`${API_URL}/api/users/profile`, {
    headers: { Authorization: `Bearer ${token}` },
  })

  if (!res.ok) {
    throw new Error(`Profile fetch failed (${res.status})`)
  }

  return res.json()
}

/**
 * Register-or-login helper: tries to register, then logs in.
 * Useful for ensuring a test user exists and obtaining valid tokens.
 */
export async function ensureUser(
  payload: RegisterPayload & { password: string }
): Promise<AuthTokens> {
  await registerUser(payload).catch(() => {
    /* user may already exist */
  })
  return loginUser({ email: payload.email, password: payload.password })
}

/** Seeded admin credentials */
export const ADMIN_CREDENTIALS = {
  email: 'admin@slate.edu',
  password: 'Admin@12345',
} as const

/** Generate a unique test email to avoid collisions between parallel runs */
export function uniqueEmail(prefix: string): string {
  const ts = Date.now()
  const rand = Math.random().toString(36).slice(2, 6)
  return `${prefix}_${ts}_${rand}@test.slate.edu`
}
