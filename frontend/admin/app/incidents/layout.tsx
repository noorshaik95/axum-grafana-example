'use client'

import { AdminShell } from '@/components/layout/admin-shell'

export default function IncidentsLayout({ children }: { children: React.ReactNode }) {
  return <AdminShell>{children}</AdminShell>
}
