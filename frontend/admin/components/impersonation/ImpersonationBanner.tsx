'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { AlertTriangle } from 'lucide-react'

interface ImpersonationUser {
  name: string
  tenantName: string
}

export function ImpersonationBanner() {
  const [user, setUser] = useState<ImpersonationUser | null>(null)
  const router = useRouter()

  useEffect(() => {
    const stored = localStorage.getItem('impersonation_user')
    if (stored) {
      try {
        setUser(JSON.parse(stored) as ImpersonationUser)
      } catch {
        setUser(null)
      }
    }
  }, [])

  if (!user) return null

  const endImpersonation = () => {
    localStorage.removeItem('impersonation_token')
    localStorage.removeItem('impersonation_user')
    router.push('/dashboard')
    router.refresh()
  }

  return (
    <div className="fixed top-0 left-0 right-0 z-50 flex items-center justify-center gap-3 bg-yellow-500 px-4 py-2 text-black">
      <AlertTriangle className="h-4 w-4 shrink-0" />
      <span className="text-sm font-medium">
        Impersonating {user.name} ({user.tenantName})
      </span>
      <Button
        size="sm"
        variant="outline"
        className="ml-2 h-7 border-black/30 bg-transparent text-black hover:bg-yellow-600"
        onClick={endImpersonation}
      >
        Return to Admin
      </Button>
    </div>
  )
}
