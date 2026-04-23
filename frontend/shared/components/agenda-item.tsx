import * as React from 'react'
import { cn } from '../utils/index'

export type AgendaUrgency = 'due-soon' | 'overdue' | 'upcoming'

export interface AgendaItemProps extends React.HTMLAttributes<HTMLDivElement> {
  icon?: React.ReactNode
  title: string
  urgency: AgendaUrgency
  /** Human-readable relative time, e.g. "in 2 hours", "3 days ago". */
  timeUntil: string
  subtitle?: string
}

const urgencyClasses: Record<AgendaUrgency, { chip: string; label: string }> = {
  'due-soon': { chip: 'bg-amber-100 text-amber-700', label: 'Due soon' },
  overdue: { chip: 'bg-red-100 text-red-700', label: 'Overdue' },
  upcoming: { chip: 'bg-forest-100 text-forest-700', label: 'Upcoming' },
}

export function AgendaItem({
  icon,
  title,
  urgency,
  timeUntil,
  subtitle,
  className,
  ...rest
}: AgendaItemProps) {
  const u = urgencyClasses[urgency]
  return (
    <div
      data-testid="agenda-item"
      data-urgency={urgency}
      className={cn(
        'flex items-center gap-3 rounded-card border border-warm-200/70 bg-paper px-3 py-2.5',
        className
      )}
      {...rest}
    >
      {icon ? (
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-forest-100 text-forest-700">
          {icon}
        </div>
      ) : null}
      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-sm font-medium text-warm-900">{title}</span>
        {subtitle ? <span className="truncate text-xs text-warm-700">{subtitle}</span> : null}
      </div>
      <span
        className={cn(
          'shrink-0 rounded-badge px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide',
          u.chip
        )}
      >
        {u.label}
      </span>
      <span className="shrink-0 text-xs tabular-nums text-warm-700">{timeUntil}</span>
    </div>
  )
}

AgendaItem.displayName = 'AgendaItem'
