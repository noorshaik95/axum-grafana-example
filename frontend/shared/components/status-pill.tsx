import * as React from 'react'
import { cn } from '../utils/index'

export type StatusPillTone = 'green' | 'amber' | 'red' | 'gray' | 'blue'

export interface StatusPillProps extends React.HTMLAttributes<HTMLSpanElement> {
  tone: StatusPillTone
  label: string
  count?: number
}

const toneClasses: Record<StatusPillTone, { dot: string; bg: string; text: string }> = {
  green: { dot: 'bg-forest-500', bg: 'bg-forest-100', text: 'text-forest-700' },
  amber: { dot: 'bg-amber-500', bg: 'bg-amber-100', text: 'text-amber-700' },
  red: { dot: 'bg-red-500', bg: 'bg-red-100', text: 'text-red-700' },
  gray: { dot: 'bg-warm-300', bg: 'bg-warm-100', text: 'text-warm-700' },
  blue: { dot: 'bg-blue-500', bg: 'bg-blue-100', text: 'text-blue-700' },
}

export function StatusPill({ tone, label, count, className, ...rest }: StatusPillProps) {
  const t = toneClasses[tone]
  return (
    <span
      data-testid="status-pill"
      data-tone={tone}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-badge px-2.5 py-0.5 text-xs font-medium',
        t.bg,
        t.text,
        className
      )}
      {...rest}
    >
      <span aria-hidden className={cn('h-1.5 w-1.5 rounded-full', t.dot)} />
      <span>{label}</span>
      {typeof count === 'number' ? (
        <span className="ml-1 rounded-full bg-white/60 px-1.5 text-[10px] font-semibold leading-4">
          {count}
        </span>
      ) : null}
    </span>
  )
}

StatusPill.displayName = 'StatusPill'
