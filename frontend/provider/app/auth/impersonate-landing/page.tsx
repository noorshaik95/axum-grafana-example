'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Loader2 } from 'lucide-react'

export default function ImpersonateLandingPage() {
  const router = useRouter()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const hash = window.location.hash.slice(1) // strip leading '#'

    if (!hash) {
      setError('Impersonation link is invalid or expired.')
      return
    }

    let token: string | null = null
    let expiresAt: string | null = null
    let tenantSlug: string | null = null

    try {
      const params = new URLSearchParams(hash)
      token = params.get('token')
      expiresAt = params.get('expires_at')
      tenantSlug = params.get('tenant_slug')
    } catch {
      setError('Impersonation link is invalid or expired.')
      return
    }

    if (!token || !expiresAt || !tenantSlug) {
      setError('Impersonation link is invalid or expired.')
      return
    }

    // Compute max-age from expires_at (seconds since epoch)
    const maxAge = Math.max(0, Number(expiresAt) - Math.floor(Date.now() / 1000))

    // Store token in localStorage for the API client interceptor
    localStorage.setItem('slate_token', token)

    // Set the cookie so the Next.js middleware allows access to protected routes
    document.cookie = `slate_token=${token}; path=/; max-age=${maxAge}; samesite=lax`

    router.replace('/teach')
  }, [router])

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
        <div className="w-full max-w-md rounded-2xl border border-red-200 bg-red-50 p-8 text-center shadow-sm">
          <p className="text-sm font-medium text-red-700">{error}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="flex flex-col items-center gap-3 text-slate-500">
        <Loader2 className="h-6 w-6 animate-spin" />
        <p className="text-sm">Starting impersonation session…</p>
      </div>
    </div>
  )
}
