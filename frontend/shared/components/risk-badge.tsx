import * as React from 'react'
import { cn } from '../utils/index'

export type RiskLevel = 'healthy' | 'slipping' | 'at_risk'

export interface RiskBadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  risk: RiskLevel
  /** Optional override for the visible label. Defaults to a human form of `risk`. */
  label?: string
}

const riskClasses: Record<RiskLevel, { cls: string; defaultLabel: string }> = {
  healthy: { cls: 'bg-forest-100 text-forest-700', defaultLabel: 'Healthy' },
  slipping: { cls: 'bg-amber-100 text-amber-700', defaultLabel: 'Slipping' },
  at_risk: { cls: 'bg-red-100 text-red-700', defaultLabel: 'At risk' },
}

export function RiskBadge({ risk, label, className, ...rest }: RiskBadgeProps) {
  const r = riskClasses[risk]
  return (
    <span
      data-testid="risk-badge"
      data-risk={risk}
      className={cn(
        'inline-flex items-center rounded-badge px-2 py-0.5 text-xs font-medium',
        r.cls,
        className
      )}
      {...rest}
    >
      {label ?? r.defaultLabel}
    </span>
  )
}

RiskBadge.displayName = 'RiskBadge'
