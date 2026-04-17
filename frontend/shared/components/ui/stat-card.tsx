import * as React from 'react'
import { cn } from '../../utils/index'

interface StatCardProps extends React.HTMLAttributes<HTMLDivElement> {
  label: string
  value: string | number
  icon?: React.ReactNode
  trend?: {
    direction: 'up' | 'down'
    value: string
  }
}

function StatCard({ label, value, icon, trend, className, ...props }: StatCardProps) {
  return (
    <div
      className={cn(
        'rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm',
        className
      )}
      {...props}
    >
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium text-[var(--color-text-muted)]">{label}</p>
        {icon && <div className="text-[var(--color-text-muted)]">{icon}</div>}
      </div>
      <div className="mt-2 flex items-baseline gap-2">
        <p className="text-3xl font-bold text-[var(--color-text)]">{value}</p>
        {trend && (
          <span
            className={cn(
              'text-xs font-medium',
              trend.direction === 'up' ? 'text-green-600' : 'text-red-600'
            )}
          >
            {trend.direction === 'up' ? '+' : '-'}
            {trend.value}
          </span>
        )}
      </div>
    </div>
  )
}

export { StatCard }
export type { StatCardProps }
