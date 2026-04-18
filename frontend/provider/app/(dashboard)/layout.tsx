'use client'

import { ProtectedRoute } from '@/components/auth/protected-route'
import { InstructorTopbar } from '@/components/layout/instructor-topbar'

export default function DashboardGroupLayout({ children }: { children: React.ReactNode }) {
  return (
    <ProtectedRoute>
      <div className="min-h-screen flex flex-col" style={{ background: '#fbfaf5' }}>
        <InstructorTopbar />
        <main className="flex-1 px-6 py-6 max-w-7xl mx-auto w-full">{children}</main>
      </div>
    </ProtectedRoute>
  )
}
