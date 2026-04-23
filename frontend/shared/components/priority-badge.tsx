import * as React from 'react'
import { cn } from '../utils/index'

export type Priority = 'P0' | 'P1' | 'P2' | 'P3' | 'P4'

export interface PriorityBadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  priority: Priority
  label?: string
}

const priorityClasses: Record<Priority, string> = {
  P0: 'bg-red-500 text-white',
  P1: 'bg-red-100 text-red-700',
  P2: 'bg-amber-100 text-amber-700',
  P3: 'bg-warm-100 text-warm-700',
  P4: 'bg-blue-100 text-blue-700',
}

export function PriorityBadge({ priority, label, className, ...rest }: PriorityBadgeProps) {
  return (
    <span
      data-testid="priority-badge"
      data-priority={priority}
      className={cn(
        'inline-flex items-center rounded-badge px-2 py-0.5 text-xs font-semibold uppercase tracking-wide',
        priorityClasses[priority],
        className
      )}
      {...rest}
    >
      {priority}
      {label ? <span className="ml-1.5 font-normal normal-case">{label}</span> : null}
    </span>
  )
}

PriorityBadge.displayName = 'PriorityBadge'
