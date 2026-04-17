import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

/**
 * Badge component variants for Slate Glass design system
 * Semantic colors: red for urgent/error, amber for warning, blue for success/info
 */
const badgeVariants = cva(
  'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
  {
    variants: {
      variant: {
        // Default: blue accent for success/info
        default: 'border-transparent bg-blue-500 text-white hover:bg-blue-600',
        // Secondary: slate for neutral
        secondary:
          'border-transparent bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700',
        // Destructive: red for urgent/error/overdue
        destructive: 'border-transparent bg-red-500 text-white hover:bg-red-600',
        // Outline: bordered style
        outline: 'text-foreground border-border',
        // Success: blue (same as default in Slate Glass theme)
        success: 'border-transparent bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
        // Warning: amber for medium priority/caution
        warning:
          'border-transparent bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <div className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export { Badge, badgeVariants };
