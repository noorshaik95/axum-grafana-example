'use client'

import { MessageSquare } from 'lucide-react'

export default function CommunicationsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Communications</h1>
        <p className="text-slate-500 mt-1">Announcements, messages, and discussions.</p>
      </div>

      <div className="rounded-xl border border-slate-200 bg-white p-16 text-center shadow-sm">
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-indigo-50 mb-5">
          <MessageSquare className="h-8 w-8 text-indigo-600" />
        </div>
        <h2 className="text-lg font-semibold text-slate-900">Coming soon</h2>
        <p className="mt-2 text-sm text-slate-500 max-w-sm mx-auto">
          Communications tools are under development. You will be able to post announcements,
          respond to student questions, and send direct messages.
        </p>
      </div>
    </div>
  )
}
