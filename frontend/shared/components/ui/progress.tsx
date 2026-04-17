'use client'

import * as React from 'react'
import * as ProgressPrimitive from '@radix-ui/react-progress'
import { cn } from '../../utils/index'

const Progress = React.forwardRef<
  React.ComponentRef<typeof ProgressPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof ProgressPrimitive.Root> & {
    label?: string
    showValue?: boolean
  }
>(({ className, value, label, showValue, ...props }, ref) => (
  <div className="w-full">
    {(label || showValue) && (
      <div className="flex items-center justify-between mb-1.5">
        {label && <span className="text-sm font-medium text-[var(--color-text)]">{label}</span>}
        {showValue && (
          <span className="text-sm text-[var(--color-text-muted)]">{Math.round(value || 0)}%</span>
        )}
      </div>
    )}
    <ProgressPrimitive.Root
      ref={ref}
      className={cn(
        'relative h-2 w-full overflow-hidden rounded-full bg-[var(--color-bg-muted)]',
        className
      )}
      {...props}
    >
      <ProgressPrimitive.Indicator
        className="h-full rounded-full bg-[var(--color-primary)] transition-all duration-300"
        style={{ width: `${value || 0}%` }}
      />
    </ProgressPrimitive.Root>
  </div>
))
Progress.displayName = 'Progress'

export { Progress }
