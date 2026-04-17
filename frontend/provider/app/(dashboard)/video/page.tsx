'use client'

import { Button } from '@/components/ui/button'
import { Plus } from 'lucide-react'
import Link from 'next/link'
import { SessionManager } from '@/components/video/SessionManager'

export default function VideoSessionsPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Video Sessions</h1>
          <p className="text-slate-500 mt-1">Schedule and manage live class sessions.</p>
        </div>
        <Button asChild>
          <Link href="/video/new">
            <Plus className="h-4 w-4 mr-2" />
            Schedule Session
          </Link>
        </Button>
      </div>

      <SessionManager />
    </div>
  )
}
