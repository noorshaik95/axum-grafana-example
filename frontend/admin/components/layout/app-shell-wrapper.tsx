'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Bell, LogOut } from 'lucide-react'
import { AppShell } from '../../../shared/components/app-shell/app-shell'
import { TabNav, type TabItem } from '../../../shared/components/app-shell/tab-nav'
import type { NowBarChip } from '../../../shared/components/app-shell/now-bar'
import authService from '@/lib/api/auth'
import { useAdminProfile } from '@/lib/hooks/use-admin-queries'
import { ImpersonationBanner } from '@/components/impersonation/ImpersonationBanner'

const ADMIN_TABS: TabItem[] = [
  { label: 'Overview', href: '/ops' },
  // TODO(wave-4): wire `count` to live incident/tenant totals once
  //   `/api/tenants` count + `/api/incidents?status=open` count land.
  { label: 'Schools', href: '/schools' },
  { label: 'People', href: '/iam/users' },
  { label: 'Money', href: '/billing' },
  { label: 'Support', href: '/incidents' },
  { label: 'Platform', href: '/flags' },
  { label: 'Audit', href: '/iam/audit' },
]

// TODO(wave-4): replace with chips computed from real endpoints —
//   `/api/incidents?status=open` (P1 chip), `/api/tenants?status=provisioning`
//   (onboarding chip), `/api/status` (services green chip).
const ADMIN_NOW_CHIPS: NowBarChip[] = [
  { id: 'p1', label: '1 P1 open · 42 min', href: '/incidents' },
  {
    id: 'stanford',
    label: 'Stanford · grade export fails',
    tone: 'amber',
    href: '/schools',
  },
  { id: 'berkeley', label: 'Berkeley onboarding · 67%', href: '/onboarding' },
  { id: 'nyu', label: 'NYU upgraded · +$8k', href: '/schools' },
  { id: 'services', label: 'all services green', href: '/status' },
]

export function AppShellWrapper({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const { data: profile } = useAdminProfile()

  const initials = (() => {
    const name = profile?.fullName ?? profile?.email ?? ''
    const parts = name.trim().split(/\s+/).filter(Boolean)
    if (parts.length >= 2) {
      return `${parts[0].charAt(0)}${parts[1].charAt(0)}`.toUpperCase()
    }
    if (parts.length === 1) {
      return parts[0].slice(0, 2).toUpperCase()
    }
    return 'AD'
  })()

  const handleLogout = async () => {
    await authService.logout()
    router.push('/login')
  }

  return (
    <AppShell
      banner={<ImpersonationBanner />}
      brand={
        <Link
          href="/ops"
          className="flex items-center gap-2 font-display text-lg tracking-tight text-forest-800"
        >
          <span
            aria-hidden="true"
            className="flex h-7 w-7 items-center justify-center rounded-md bg-gradient-to-br from-forest-500 to-forest-800 text-xs font-bold text-white"
          >
            S
          </span>
          <span>Slate Admin</span>
        </Link>
      }
      centerNav={<TabNav tabs={ADMIN_TABS} />}
      right={
        <>
          <button
            type="button"
            aria-label="Open command palette"
            className="hidden rounded-full px-2.5 py-1 text-xs font-medium text-warm-700 ring-1 ring-warm-200 hover:bg-warm-50 sm:inline-flex"
          >
            ⌘K
          </button>
          <button
            type="button"
            aria-label="Notifications"
            className="relative rounded-full p-1.5 text-warm-700 hover:bg-warm-50"
          >
            <Bell className="h-4 w-4" />
            <span
              aria-hidden="true"
              className="absolute right-1 top-1 h-1.5 w-1.5 rounded-full bg-red-500"
            />
          </button>
          <div
            aria-label="Admin profile"
            className="flex h-8 w-8 items-center justify-center rounded-full bg-forest-100 text-xs font-medium text-forest-700"
          >
            {initials}
          </div>
          <button
            type="button"
            onClick={handleLogout}
            aria-label="Logout"
            className="rounded-full p-1.5 text-warm-700 hover:bg-warm-50"
          >
            <LogOut className="h-4 w-4" />
          </button>
        </>
      }
      nowChips={ADMIN_NOW_CHIPS}
      nowLabel="Now"
    >
      {children}
    </AppShell>
  )
}
