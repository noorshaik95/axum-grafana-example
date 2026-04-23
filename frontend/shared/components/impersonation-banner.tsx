'use client'

import * as React from 'react'
import { cn } from '../utils/index'
import { Button } from './ui/button'

export interface ImpersonationBannerProps extends React.HTMLAttributes<HTMLDivElement> {
  /** Email (or handle) of the user being impersonated. */
  email: string
  /** Optional tenant/school name shown in parens. */
  tenantName?: string
  /** Called when the admin clicks "End session". Consumer clears tokens + redirects. */
  onEnd: () => void
  /** Label for the end button (default: "End session"). */
  endLabel?: string
}

function AlertIcon({ className }: { className?: string }) {
  return (
    <svg
      className={cn('h-4 w-4', className)}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z" />
      <line x1="12" y1="9" x2="12" y2="13" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </svg>
  )
}

/**
 * Sticky yellow ribbon shown above the TopBar while an admin is impersonating
 * another user. Moved from `admin/components/impersonation/` so any portal
 * (admin, tenant) can render it from shared.
 */
export function ImpersonationBanner({
  email,
  tenantName,
  onEnd,
  endLabel = 'End session',
  className,
  ...rest
}: ImpersonationBannerProps) {
  return (
    <div
      role="status"
      data-testid="impersonation-banner"
      className={cn(
        'fixed top-0 left-0 right-0 z-50 flex items-center justify-center gap-3 bg-amber-400 px-4 py-2 text-warm-900',
        className
      )}
      {...rest}
    >
      <AlertIcon className="shrink-0" />
      <span className="text-sm font-medium">
        Impersonating {email}
        {tenantName ? ` (${tenantName})` : ''}
      </span>
      <Button
        size="sm"
        variant="outline"
        className="ml-2 h-7 border-warm-900/30 bg-transparent text-warm-900 hover:bg-amber-500"
        onClick={onEnd}
      >
        {endLabel}
      </Button>
    </div>
  )
}

ImpersonationBanner.displayName = 'ImpersonationBanner'
