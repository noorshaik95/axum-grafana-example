'use client'

import { ProtectedRoute } from '@/components/auth/protected-route'
import { InstructorShell } from '@/components/layout/instructor-shell'

export default function DashboardGroupLayout({ children }: { children: React.ReactNode }) {
  return (
    <ProtectedRoute>
      <InstructorShell>{children}</InstructorShell>
    </ProtectedRoute>
  )
}
