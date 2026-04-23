import apiClient from './client'

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  accessToken: string
  refreshToken: string
  expiresAtUnixMs?: string
  user: {
    id: string
    email: string
    fullName?: string
    roles: string[]
  }
}

export const authService = {
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await apiClient.post<LoginResponse>('/admin/auth/login', credentials)
    if (response.data.accessToken) {
      // admin's axios client (lib/api/client.ts) reads `admin_token`.
      localStorage.setItem('admin_token', response.data.accessToken)
      localStorage.setItem('admin_auth_token', response.data.accessToken)
      // shared fetch client (shared/lib/api/client.ts) reads `slate_token`.
      // Several admin pages use that client (platform stats, incidents,
      // flags, audit), so without this key their requests go out unauthed.
      localStorage.setItem('slate_token', response.data.accessToken)
      // middleware.ts runs on the server and can only see cookies, not
      // localStorage. Mirror the token into a cookie so the auth guard
      // lets post-login navigation through instead of bouncing to /login.
      const maxAge = 60 * 60 * 8 // 8h; matches admin-auth access token TTL
      document.cookie = `slate_token=${response.data.accessToken}; Path=/; Max-Age=${maxAge}; SameSite=Lax`
    }
    // Persist the admin user object so `useAdminProfile` can render the
    // top-nav avatar/name without a network call. Admin JWTs are issued by
    // admin-auth-service and carry a user_id that does NOT exist in
    // user-auth's `users` table, so `GET /api/users/profile` 502s (R1).
    // Admin portal must never hit that shared endpoint.
    if (response.data.user) {
      localStorage.setItem('admin_user', JSON.stringify(response.data.user))
    }
    return response.data
  },

  async logout(): Promise<void> {
    try {
      await apiClient.post('/admin/auth/logout')
    } finally {
      localStorage.removeItem('admin_auth_token')
      localStorage.removeItem('admin_token')
      localStorage.removeItem('slate_token')
      localStorage.removeItem('admin_user')
      document.cookie = 'slate_token=; Path=/; Max-Age=0; SameSite=Lax'
    }
  },

  async validateToken(): Promise<boolean> {
    try {
      const response = await apiClient.post('/admin/auth/validate')
      return response.data.valid
    } catch {
      return false
    }
  },

  getToken(): string | null {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('admin_auth_token')
    }
    return null
  },

  isAuthenticated(): boolean {
    return !!this.getToken()
  },
}

export default authService
