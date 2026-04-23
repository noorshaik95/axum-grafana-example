'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import authService from '../../lib/api/auth'

/**
 * Admin-only ProtectedRoute. Uses admin-auth profile (via admin axios
 * client at `/admin/auth/profile`) instead of the shared user-auth
 * `/api/users/profile` which 502s for admin JWTs (R1). Role data is
 * sourced from the persisted `admin_user` localStorage key written at
 * login — no network call on the happy path.
 */
interface ProtectedRouteProps {
  children: React.ReactNode
}

type RoleLike = string | { name?: string }

function rolesFromStorage(): RoleLike[] {
  if (typeof window === 'undefined') return []
  const raw = window.localStorage.getItem('admin_user')
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw) as { roles?: RoleLike[] }
    return parsed.roles ?? []
  } catch {
    return []
  }
}

function isAdminRole(roles: RoleLike[]): boolean {
  return roles.some((r) => {
    const name = typeof r === 'string' ? r : r?.name
    return name === 'admin' || name === 'superadmin'
  })
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const [isAuthorized, setIsAuthorized] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const router = useRouter()

  useEffect(() => {
    const checkAuth = async () => {
      if (typeof window === 'undefined') return

      const token =
        localStorage.getItem('admin_auth_token') ||
        localStorage.getItem('admin_token') ||
        localStorage.getItem('slate_token')
      if (!token) {
        router.push('/login')
        return
      }

      // Prefer the cached admin_user — avoids any profile network call.
      const cachedRoles = rolesFromStorage()
      if (cachedRoles.length > 0) {
        if (!isAdminRole(cachedRoles)) {
          router.push('/login')
          return
        }
        setIsAuthorized(true)
        setIsLoading(false)
        return
      }

      // Fallback: hit admin-auth profile (admin axios client). Never the
      // shared user-auth /api/users/profile route.
      try {
        const profile = await authService.getProfile()
        const roles = (profile.roles ?? []) as RoleLike[]
        if (!isAdminRole(roles)) {
          router.push('/login')
          return
        }
        setIsAuthorized(true)
      } catch {
        localStorage.removeItem('admin_auth_token')
        localStorage.removeItem('admin_token')
        localStorage.removeItem('slate_token')
        localStorage.removeItem('admin_user')
        router.push('/login')
      } finally {
        setIsLoading(false)
      }
    }

    checkAuth()
  }, [router])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    )
  }

  if (!isAuthorized) {
    return null
  }

  return <>{children}</>
}
