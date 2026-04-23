'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Bell, LogOut } from 'lucide-react'
import { useMemo } from 'react'
import { AppShell } from '../../../shared/components/app-shell/app-shell'
import { TabNav, type TabItem } from '../../../shared/components/app-shell/tab-nav'
import type { NowBarChip } from '../../../shared/components/app-shell/now-bar'
import { useProfile } from '../../../shared/lib/api/hooks'
import { getCurrentClaims } from '../../lib/api/client'

const PROVIDER_TABS: TabItem[] = [
  { label: 'Teach', href: '/teach' },
  { label: 'Courses', href: '/courses' },
  // TODO(CARRYOVER §1a / task #29): wire `count` to the real queue
  //   aggregate once gateway-proxy-expert exposes an instructor-scoped
  //   count. /teach approximates from past-due assignments today.
  { label: 'Grade', href: '/grade' },
  { label: 'Roster', href: '/roster' },
  { label: 'Office hours', href: '/office-hours' },
  { label: 'Discussion', href: '/discussion' },
]

// TODO(wave-4): replace with chips computed from real endpoints —
//   `/api/grading/queue/count` (pending chip), `/api/video/lectures/next`
//   (next lecture chip), `/api/scheduling/instructor-day` (OH chip).
const PROVIDER_NOW_CHIPS: NowBarChip[] = [
  { id: 'queue', label: '47 in queue · PS4', tone: 'amber', href: '/grade' },
  { id: 'lecture', label: 'Lecture 7 · 10:00', href: '/teach' },
  { id: 'oh', label: 'OH · 4 booked · 2pm', href: '/office-hours' },
]

export function AppShellWrapper({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  // useProfile may 404 today (#57). Don't block shell rendering on it —
  // fall back to JWT claims so initials + profile UI render offline.
  const { data: profile } = useProfile()
  const claims = useMemo(() => getCurrentClaims(), [])

  const firstName = profile?.firstName ?? claims.firstName ?? ''
  const lastName = profile?.lastName ?? claims.lastName ?? ''
  const initials = `${firstName[0] ?? ''}${lastName[0] ?? ''}`.toUpperCase() || 'IN'

  const handleLogout = () => {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('slate_token')
      localStorage.removeItem('slate_refresh_token')
      document.cookie = 'slate_token=; path=/; max-age=0'
    }
    router.push('/login')
  }

  return (
    <AppShell
      brand={
        <Link
          href="/teach"
          className="flex items-center gap-2 font-display text-lg tracking-tight text-forest-800"
        >
          <span
            aria-hidden="true"
            className="flex h-7 w-7 items-center justify-center rounded-md bg-gradient-to-br from-forest-500 to-forest-800 text-xs font-bold text-white"
          >
            S
          </span>
          <span>Slate</span>
        </Link>
      }
      centerNav={<TabNav tabs={PROVIDER_TABS} />}
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
              className="absolute right-1 top-1 h-1.5 w-1.5 rounded-full bg-amber-500"
            />
          </button>
          <div
            aria-label="Instructor profile"
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
      nowChips={PROVIDER_NOW_CHIPS}
      nowLabel="Now"
    >
      {children}
    </AppShell>
  )
}
