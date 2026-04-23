'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { decodeJwtClaims, hasInstructorRole } from '../../lib/api/client'

interface ProtectedRouteProps {
  children: React.ReactNode
}

/**
 * T3-R1 (#57): previously this component blocked layout mount on a successful
 * GET /api/users/profile response. That endpoint 404s today for instructor
 * JWTs (seed / gateway mismatch), so 18 dashboard pages were stuck spinning
 * on every load.
 *
 * We now authorize purely from the JWT claims (same pattern student-fe uses
 * per task #46). The backend still validates the token on every downstream
 * request — this shell-level decode is an optimistic gate that lets the UI
 * render immediately. If the token is missing/expired or lacks an instructor
 * role, we redirect to /login exactly as before.
 */
export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const [status, setStatus] = useState<'checking' | 'authorized' | 'redirecting'>('checking')
  const router = useRouter()

  useEffect(() => {
    if (typeof window === 'undefined') return

    const token = localStorage.getItem('slate_token')
    if (!token) {
      setStatus('redirecting')
      router.push('/login')
      return
    }

    const claims = decodeJwtClaims(token)
    if (!claims.userId || !hasInstructorRole(claims)) {
      localStorage.removeItem('slate_token')
      setStatus('redirecting')
      router.push('/login')
      return
    }

    setStatus('authorized')
  }, [router])

  if (status === 'checking') {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-12 w-12 animate-spin rounded-full border-b-2 border-forest-600" />
      </div>
    )
  }

  if (status !== 'authorized') return null

  return <>{children}</>
}
