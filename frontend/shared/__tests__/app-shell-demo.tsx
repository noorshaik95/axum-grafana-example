/**
 * AppShell visual demo + smoke test.
 *
 * Composes TopBar + TabNav + NowBar with the exact chip/tab set shown in the
 * admin reference canvas: horizontal tabs ("Overview · Schools 42 · People ·
 * Money · Support 3 · Platform · Audit") atop the cream page background, and a
 * forest-green Now Bar with "1 P1 open · 42 min", an amber "Stanford · grade
 * export fails", "Berkeley onboarding · 67%", "NYU upgraded · +$8k", and
 * "all services green" chips.
 *
 * The test asserts the shell renders its three structural pieces and the
 * reference chips/tabs are present — serves both as a live demo and a
 * regression guard on the shell's composition.
 */

import * as React from 'react'
import { render, screen } from '@testing-library/react'
import { AppShell } from '../components/app-shell/app-shell'
import { TabNav, type TabItem } from '../components/app-shell/tab-nav'
import type { NowBarChip } from '../components/app-shell/now-bar'

jest.mock('next/link', () => ({
  __esModule: true,
  default: ({ href, children, ...rest }: any) =>
    React.createElement('a', { href, ...rest }, children),
}))

jest.mock('next/navigation', () => ({
  usePathname: () => '/overview',
}))

export const ADMIN_TABS: TabItem[] = [
  { label: 'Overview', href: '/overview' },
  { label: 'Schools', href: '/schools', count: 42 },
  { label: 'People', href: '/people' },
  { label: 'Money', href: '/money' },
  { label: 'Support', href: '/support', count: 3, badgeTone: 'amber' },
  { label: 'Platform', href: '/platform' },
  { label: 'Audit', href: '/audit' },
]

export const ADMIN_NOW_CHIPS: NowBarChip[] = [
  { label: '1 P1 open · 42 min', href: '/incidents/P1' },
  { label: 'Stanford · grade export fails', tone: 'amber', href: '/schools/stanford' },
  { label: 'Berkeley onboarding · 67%', href: '/schools/berkeley' },
  { label: 'NYU upgraded · +$8k', href: '/schools/nyu' },
  { label: 'all services green', tone: 'neutral', href: '/status' },
]

export function AdminAppShellDemo() {
  return (
    <AppShell
      brand={
        <a href="/" className="font-display text-lg tracking-tight text-forest-800">
          Slate
        </a>
      }
      centerNav={<TabNav tabs={ADMIN_TABS} />}
      right={
        <>
          <button
            aria-label="Open command palette"
            className="rounded-button px-2.5 py-1 text-xs font-medium text-warm-700 ring-1 ring-warm-200 hover:bg-warm-50"
          >
            ⌘K
          </button>
          <button aria-label="Notifications" className="rounded-full p-1.5 hover:bg-warm-50">
            <span aria-hidden="true">🔔</span>
          </button>
          <div
            aria-label="Admin profile"
            className="flex h-8 w-8 items-center justify-center rounded-full bg-forest-100 text-xs font-medium text-forest-700"
          >
            AD
          </div>
        </>
      }
      nowChips={ADMIN_NOW_CHIPS}
      nowLabel="Now"
    >
      <div className="mt-4 text-sm text-warm-700">
        Placeholder content — concrete pages plug in under the shell.
      </div>
    </AppShell>
  )
}

describe('AppShell demo (admin reference)', () => {
  it('renders the shell with TopBar, NowBar, all reference tabs, and all reference chips', () => {
    render(<AdminAppShellDemo />)

    expect(screen.getByTestId('top-bar')).toBeInTheDocument()
    expect(screen.getByTestId('now-bar')).toBeInTheDocument()
    expect(screen.getByRole('main')).toBeInTheDocument()

    ADMIN_TABS.forEach((t) => {
      expect(screen.getByRole('link', { name: new RegExp(t.label, 'i') })).toBeInTheDocument()
    })

    ADMIN_NOW_CHIPS.forEach((c) => {
      expect(screen.getByText(c.label)).toBeInTheDocument()
    })
  })
})
