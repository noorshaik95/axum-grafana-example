'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { auth } from '../../../shared/lib/api'
import type { User } from '../../../shared/lib/api/types'

interface ProtectedRouteProps {
  children: React.ReactNode
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const [isAuthorized, setIsAuthorized] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const router = useRouter()

  useEffect(() => {
    const checkAuth = async () => {
      if (typeof window === 'undefined') return

      const token = localStorage.getItem('slate_token')
      if (!token) {
        router.push('/login')
        return
      }

      try {
        const profile: User = await auth.getProfile()
        const isInstructor = profile.roles.some(
          (r) => r.name === 'instructor' || r.name === 'admin' || r.name === 'superadmin'
        )

        if (!isInstructor) {
          router.push('/login')
          return
        }

        setIsAuthorized(true)
      } catch {
        localStorage.removeItem('slate_token')
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
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600" />
      </div>
    )
  }

  if (!isAuthorized) {
    return null
  }

  return <>{children}</>
}
