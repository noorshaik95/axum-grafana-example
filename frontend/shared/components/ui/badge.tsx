import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../utils/index'

const badgeVariants = cva(
  'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors',
  {
    variants: {
      variant: {
        default: 'bg-[var(--color-primary-light)] text-[var(--color-primary)]',
        secondary: 'bg-gray-100 text-gray-600',
        success: 'bg-[var(--color-success-light)] text-green-700',
        warning: 'bg-[var(--color-warning-light)] text-amber-700',
        error: 'bg-[var(--color-error-light)] text-red-700',
        info: 'bg-[var(--color-info-light)] text-blue-700',
        outline: 'border border-[var(--color-border)] text-[var(--color-text-muted)]',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />
}

export { Badge, badgeVariants }
