'use client'

import * as React from 'react'
import { cn } from '../../utils/index'

export interface TopBarProps {
  /** Wordmark / brand slot. Usually a <Link href="/"> with the Slate logo + label. */
  brand: React.ReactNode
  /** Center slot — typically <TabNav /> for the portal's primary navigation. */
  centerNav?: React.ReactNode
  /** Right slot — Cmd+K trigger, NotificationBell, avatar menu. */
  right?: React.ReactNode
  className?: string
}

/**
 * Sticky, blurred cream top bar. Content scrolls behind it thanks to
 * `backdrop-blur-md bg-cream/80`. Height fixed at 56px (h-14).
 */
export function TopBar({ brand, centerNav, right, className }: TopBarProps) {
  return (
    <header
      role="banner"
      data-testid="top-bar"
      className={cn(
        'sticky top-0 z-40 w-full border-b border-warm-200/80 bg-cream/80 backdrop-blur-md',
        'supports-[backdrop-filter]:bg-cream/70',
        className
      )}
    >
      <div className="mx-auto flex h-14 max-w-[1400px] items-center gap-4 px-6">
        <div className="flex shrink-0 items-center gap-3 text-warm-900">{brand}</div>
        <div className="flex min-w-0 flex-1 justify-center">{centerNav}</div>
        <div className="flex shrink-0 items-center gap-2 text-warm-700">{right}</div>
      </div>
    </header>
  )
}

TopBar.displayName = 'TopBar'
