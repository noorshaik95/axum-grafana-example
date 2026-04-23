import apiClient from './client'

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  user: {
    id: string
    email: string
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
    const response = await apiClient.post('/admin/auth/login', credentials)
    if (response.data.access_token) {
      localStorage.setItem('admin_auth_token', response.data.access_token)
    }
    return response.data
  },

  async logout(): Promise<void> {
    try {
      await apiClient.post('/admin/auth/logout')
    } finally {
      localStorage.removeItem('admin_auth_token')
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
