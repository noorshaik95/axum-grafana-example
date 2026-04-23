'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

/**
 * Admin-only ProtectedRoute. Role data is sourced from the persisted
 * `admin_user` localStorage key written at login. No profile network
 * call: admin-auth proto has no GetProfile RPC and the gateway has no
 * `/api/admin/auth/profile` route (#56). If the localStorage payload is
 * missing, we bounce to `/login` — the user re-auths and the login
 * handler repopulates `admin_user` fresh.
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

function clearAuthStorage() {
  localStorage.removeItem('admin_auth_token')
  localStorage.removeItem('admin_token')
  localStorage.removeItem('slate_token')
  localStorage.removeItem('admin_user')
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const [isAuthorized, setIsAuthorized] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const router = useRouter()

  useEffect(() => {
    if (typeof window === 'undefined') return

    const token =
      localStorage.getItem('admin_auth_token') ||
      localStorage.getItem('admin_token') ||
      localStorage.getItem('slate_token')
    if (!token) {
      router.push('/login')
      return
    }

    const roles = rolesFromStorage()
    if (roles.length === 0 || !isAdminRole(roles)) {
      clearAuthStorage()
      router.push('/login')
      return
    }

    setIsAuthorized(true)
    setIsLoading(false)
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
