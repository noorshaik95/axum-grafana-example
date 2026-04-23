import * as React from 'react'
import { cn } from '../utils/index'

export interface ListRowProps extends React.HTMLAttributes<HTMLDivElement> {
  /** Optional explicit index used to alternate warm-50 / cream backgrounds. */
  index?: number
  leading?: React.ReactNode
  primary: React.ReactNode
  secondary?: React.ReactNode
  trailing?: React.ReactNode
  action?: React.ReactNode
  /** Makes the entire row feel clickable (no navigation; wire up with onClick/href in parent). */
  interactive?: boolean
}

/**
 * A full-width row used in inboxes, grading queues, rosters. Backgrounds
 * alternate warm-50 / cream. If `index` is provided, parity is controlled
 * explicitly; otherwise the consumer can style the row via nth-child selectors.
 */
export function ListRow({
  index,
  leading,
  primary,
  secondary,
  trailing,
  action,
  interactive,
  className,
  ...rest
}: ListRowProps) {
  const bg = typeof index === 'number' ? (index % 2 === 0 ? 'bg-cream' : 'bg-warm-50') : ''
  return (
    <div
      data-testid="list-row"
      className={cn(
        'flex items-center gap-3 border-b border-warm-200/60 px-4 py-3 text-warm-900 last:border-b-0',
        bg,
        interactive && 'cursor-pointer hover:bg-forest-50/40',
        className
      )}
      {...rest}
    >
      {leading ? (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center text-forest-600">
          {leading}
        </div>
      ) : null}
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="truncate text-sm font-medium">{primary}</div>
        {secondary ? <div className="truncate text-xs text-warm-700">{secondary}</div> : null}
      </div>
      {trailing ? <div className="shrink-0">{trailing}</div> : null}
      {action ? <div className="ml-2 shrink-0">{action}</div> : null}
    </div>
  )
}

ListRow.displayName = 'ListRow'
