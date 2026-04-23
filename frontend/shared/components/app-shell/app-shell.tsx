'use client'

import * as React from 'react'
import { cn } from '../../utils/index'
import { TopBar } from './top-bar'
import { NowBar, type NowBarChip } from './now-bar'

export interface AppShellProps {
  brand: React.ReactNode
  centerNav?: React.ReactNode
  right?: React.ReactNode
  /** NowBar chips — if empty/undefined the bar auto-hides. */
  nowChips?: NowBarChip[]
  nowLabel?: string
  /** Rendered above the TopBar (ImpersonationBanner, env banner, etc). */
  banner?: React.ReactNode
  children: React.ReactNode
  className?: string
  mainClassName?: string
}

/**
 * Composes TopBar + (optional) NowBar + <main>. Every portal should wrap its
 * page tree with this shell so navigation + contextual actions stay consistent.
 */
export function AppShell({
  brand,
  centerNav,
  right,
  nowChips,
  nowLabel,
  banner,
  children,
  className,
  mainClassName,
}: AppShellProps) {
  return (
    <div className={cn('flex min-h-screen flex-col bg-cream', className)}>
      {banner ? <div className="w-full">{banner}</div> : null}
      <TopBar brand={brand} centerNav={centerNav} right={right} />
      <NowBar chips={nowChips ?? []} label={nowLabel} />
      <main
        role="main"
        className={cn('mx-auto w-full max-w-[1400px] flex-1 px-6 py-6', mainClassName)}
      >
        {children}
      </main>
    </div>
  )
}

AppShell.displayName = 'AppShell'
