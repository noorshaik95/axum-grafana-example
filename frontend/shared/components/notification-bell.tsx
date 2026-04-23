'use client'

import * as React from 'react'
import { cn } from '../utils/index'

export interface NotificationBellProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  unreadCount: number
  /** Max number before showing `99+`. */
  max?: number
}

function BellIcon({ className }: { className?: string }) {
  return (
    <svg
      className={cn('h-5 w-5', className)}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
      <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
    </svg>
  )
}

export const NotificationBell = React.forwardRef<HTMLButtonElement, NotificationBellProps>(
  function NotificationBell(
    { unreadCount, max = 99, className, 'aria-label': ariaLabel, ...rest },
    ref
  ) {
    const hasUnread = unreadCount > 0
    const display = unreadCount > max ? `${max}+` : String(unreadCount)
    return (
      <button
        ref={ref}
        type="button"
        data-testid="notification-bell"
        aria-label={
          ariaLabel ?? (hasUnread ? `Notifications, ${unreadCount} unread` : 'Notifications')
        }
        className={cn(
          'relative inline-flex h-9 w-9 items-center justify-center rounded-full text-warm-700 transition-colors hover:bg-warm-100 hover:text-warm-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-forest-500',
          className
        )}
        {...rest}
      >
        <BellIcon />
        {hasUnread ? (
          <span
            data-testid="notification-bell-badge"
            className="absolute -right-0.5 -top-0.5 inline-flex min-w-[16px] items-center justify-center rounded-full bg-forest-600 px-1 text-[10px] font-semibold leading-4 text-white ring-2 ring-cream"
          >
            {display}
          </span>
        ) : null}
      </button>
    )
  }
)

NotificationBell.displayName = 'NotificationBell'
