'use client'

import * as React from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '../../utils/index'

export type TabBadgeTone = 'neutral' | 'amber' | 'red' | 'forest' | 'blue'

export interface TabItem {
  label: string
  href: string
  count?: number
  badgeTone?: TabBadgeTone
  /** Optional leading icon for portals that want one. */
  icon?: React.ReactNode
}

export interface TabNavProps {
  tabs: TabItem[]
  /** Override active detection; default compares current pathname to `href`. */
  activeHref?: string
  className?: string
  ariaLabel?: string
}

const badgeToneClass: Record<TabBadgeTone, string> = {
  neutral: 'bg-warm-100 text-warm-700',
  amber: 'bg-amber-100 text-amber-700',
  red: 'bg-red-100 text-red-700',
  forest: 'bg-forest-100 text-forest-700',
  blue: 'bg-blue-100 text-blue-700',
}

/**
 * Horizontal tab navigation — replaces the legacy vertical sidebars.
 *
 * Active tab: forest-100 background + forest-800 text (matches the reference canvas).
 * Inactive: warm-700 text with a forest-50 hover.
 */
export function TabNav({ tabs, activeHref, className, ariaLabel = 'Primary' }: TabNavProps) {
  const pathname = usePathname() ?? ''
  const resolvedActive = activeHref ?? pathname

  const isActive = (href: string) => {
    if (resolvedActive === href) return true
    if (href === '/') return resolvedActive === '/'
    return resolvedActive.startsWith(`${href}/`)
  }

  return (
    <nav
      role="navigation"
      aria-label={ariaLabel}
      className={cn('flex items-center gap-1 overflow-x-auto', className)}
    >
      {tabs.map((tab) => {
        const active = isActive(tab.href)
        const tone = tab.badgeTone ?? 'neutral'
        return (
          <Link
            key={tab.href}
            href={tab.href}
            aria-current={active ? 'page' : undefined}
            data-active={active ? 'true' : 'false'}
            className={cn(
              'group flex items-center gap-2 whitespace-nowrap rounded-full px-3.5 py-1.5 text-sm font-medium transition-colors',
              active
                ? 'bg-forest-100 text-forest-800'
                : 'text-warm-700 hover:bg-forest-50 hover:text-forest-800'
            )}
          >
            {tab.icon ? (
              <span className="text-current" aria-hidden="true">
                {tab.icon}
              </span>
            ) : null}
            <span>{tab.label}</span>
            {typeof tab.count === 'number' ? (
              <span
                data-testid="tab-count"
                className={cn(
                  'inline-flex min-w-[1.25rem] items-center justify-center rounded-full px-1.5 py-0.5 text-[11px] font-semibold leading-none',
                  active ? 'bg-forest-600 text-white' : badgeToneClass[tone]
                )}
              >
                {tab.count}
              </span>
            ) : null}
          </Link>
        )
      })}
    </nav>
  )
}

TabNav.displayName = 'TabNav'
