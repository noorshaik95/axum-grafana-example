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

export interface User {
  id: string
  email: string
  name: string
  roles: string[]
  created_at: string
}

export const authService = {
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await apiClient.post<LoginResponse>('/admin/auth/login', credentials)
    if (response.data.accessToken) {
      localStorage.setItem('admin_auth_token', response.data.accessToken)
      // client.ts' request interceptor reads `admin_token` — keep both keys in
      // sync so authenticated requests work after login without a reload.
      localStorage.setItem('admin_token', response.data.accessToken)
      // middleware.ts runs on the server and can only see cookies, not
      // localStorage. Set slate_token cookie so the auth guard lets
      // post-login navigation through instead of bouncing back to /login.
      const maxAge = 60 * 60 * 8 // 8h; matches admin-auth access token TTL
      document.cookie = `slate_token=${response.data.accessToken}; Path=/; Max-Age=${maxAge}; SameSite=Lax`
    }
    return response.data
  },

  async logout(): Promise<void> {
    try {
      await apiClient.post('/admin/auth/logout')
    } finally {
      localStorage.removeItem('admin_auth_token')
      localStorage.removeItem('admin_token')
      document.cookie = 'slate_token=; Path=/; Max-Age=0; SameSite=Lax'
    }
  },

  async getProfile(): Promise<User> {
    const response = await apiClient.get('/admin/auth/profile')
    return response.data
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
