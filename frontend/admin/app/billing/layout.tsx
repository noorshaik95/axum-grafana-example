'use client'

import { AppShellWrapper } from '@/components/layout/app-shell-wrapper'

export default function BillingLayout({ children }: { children: React.ReactNode }) {
  return <AppShellWrapper>{children}</AppShellWrapper>
}
