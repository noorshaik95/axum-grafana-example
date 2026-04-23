import axios, { AxiosInstance, InternalAxiosRequestConfig } from 'axios'

const RAW_API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://api.slate.local'
// Gateway mounts all routes under /api/* (see config/gateway-config.yaml).
// Strip any trailing slash from the env value and append /api once so
// callers can keep writing `/tenants`, `/admin/auth/login`, etc.
const API_URL = RAW_API_URL.replace(/\/+$/, '') + '/api'

class ApiClient {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: API_URL,
      headers: {
        'Content-Type': 'application/json',
      },
      timeout: 30000,
    })

    this.client.interceptors.request.use(
      (config: InternalAxiosRequestConfig) => {
        if (typeof window !== 'undefined') {
          // Use impersonation token if active, otherwise admin token
          const impersonationToken = localStorage.getItem('impersonation_token')
          const adminToken = localStorage.getItem('admin_token')
          const token = impersonationToken || adminToken

          if (token && config.headers) {
            config.headers.Authorization = `Bearer ${token}`
          }

          // Always include tenant header for admin requests
          config.headers['X-Tenant-ID'] = 'system'
        }
        return config
      },
      (error) => Promise.reject(error)
    )

    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          if (typeof window !== 'undefined') {
            localStorage.removeItem('admin_token')
            localStorage.removeItem('impersonation_token')
            localStorage.removeItem('impersonation_user')
            window.location.href = '/login'
          }
        }
        return Promise.reject(error)
      }
    )
  }

  getInstance(): AxiosInstance {
    return this.client
  }
}

export const apiClient = new ApiClient().getInstance()
export default apiClient
