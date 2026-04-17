/**
 * Slate Glass Design System - StatCard Component
 *
 * A standardized stat card component for displaying metrics consistently
 * across all portal dashboards. Uses glass card styling with centered content.
 *
 * Features:
 * - Glass card styling with backdrop blur
 * - Label, value, and icon props
 * - Optional trend indicator with slate colors
 * - Centered content layout
 * - Responsive padding for mobile
 *
 * Requirements: 5.1, 5.2, 5.3
 */

import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../utils'

/**
 * StatCard variant styles using class-variance-authority
 *
 * Base styles include:
 * - Glassmorphism with backdrop blur
 * - Slate-colored borders
 * - Centered content layout
 * - Rounded corners using design system radius
 *
 * Requirements: 5.3
 */
export const statCardVariants = cva(
  // Base glassmorphism styling with centered content
  [
    'rounded-xl',
    'bg-card/70 dark:bg-card/50',
    'backdrop-blur-[12px]',
    'border border-card-border',
    'text-card-foreground',
    'shadow-lg',
    'transition-colors duration-200',
    'flex flex-col items-center justify-center',
    'text-center',
  ],
  {
    variants: {
      /**
       * Size variants for different stat card sizes
       */
      size: {
        default: 'p-4 sm:p-6',
        sm: 'p-3 sm:p-4',
        lg: 'p-6 sm:p-8',
      },
      /**
       * Hover behavior variants
       * No glow effects - only subtle border changes
       */
      hover: {
        default: ['hover:border-border/80', 'dark:hover:border-slate-600'],
        none: '',
        lift: [
          'hover:border-border/80',
          'dark:hover:border-slate-600',
          'hover:-translate-y-1',
          'hover:shadow-xl',
          'transition-all',
        ],
      },
    },
    defaultVariants: {
      size: 'default',
      hover: 'default',
    },
  }
)

/**
 * Trend indicator variant styles
 *
 * Uses slate colors for trend indicators:
 * - Up trend: green tones for positive
 * - Down trend: red tones for negative
 *
 * Requirements: 5.2
 */
export const trendVariants = cva(
  ['inline-flex items-center gap-1', 'text-sm font-medium', 'mt-2'],
  {
    variants: {
      direction: {
        up: 'text-green-600 dark:text-green-400',
        down: 'text-red-600 dark:text-red-400',
      },
    },
    defaultVariants: {
      direction: 'up',
    },
  }
)

/**
 * Trend indicator props
 *
 * Requirements: 5.1, 5.2
 */
export interface TrendProps {
  /**
   * Direction of the trend
   * - 'up': Positive trend (green indicator)
   * - 'down': Negative trend (red indicator)
   */
  direction: 'up' | 'down'
  /**
   * Value to display (e.g., "+12%", "-5%")
   */
  value: string
}

/**
 * StatCard component props
 *
 * Requirements: 5.1 - StatCard SHALL accept label, value, icon, and optional trend props
 */
export interface StatCardProps
  extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof statCardVariants> {
  /**
   * Label describing the statistic
   * Displayed above the value in muted text
   */
  label: string

  /**
   * The main value to display
   * Can be a string or number
   */
  value: string | number

  /**
   * Optional icon displayed above the label
   * Should be a React node (e.g., an icon component)
   */
  icon?: React.ReactNode

  /**
   * Optional trend indicator
   * Shows direction (up/down) with appropriate slate color
   *
   * Requirements: 5.2
   */
  trend?: TrendProps
}

/**
 * TrendUp Icon Component
 * Simple arrow up icon for positive trends
 */
const TrendUpIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 20 20"
    fill="currentColor"
    className={cn('w-4 h-4', className)}
    aria-hidden="true"
  >
    <path
      fillRule="evenodd"
      d="M10 17a.75.75 0 01-.75-.75V5.612L5.29 9.77a.75.75 0 01-1.08-1.04l5.25-5.5a.75.75 0 011.08 0l5.25 5.5a.75.75 0 11-1.08 1.04l-3.96-4.158V16.25A.75.75 0 0110 17z"
      clipRule="evenodd"
    />
  </svg>
)

/**
 * TrendDown Icon Component
 * Simple arrow down icon for negative trends
 */
const TrendDownIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 20 20"
    fill="currentColor"
    className={cn('w-4 h-4', className)}
    aria-hidden="true"
  >
    <path
      fillRule="evenodd"
      d="M10 3a.75.75 0 01.75.75v10.638l3.96-4.158a.75.75 0 111.08 1.04l-5.25 5.5a.75.75 0 01-1.08 0l-5.25-5.5a.75.75 0 111.08-1.04l3.96 4.158V3.75A.75.75 0 0110 3z"
      clipRule="evenodd"
    />
  </svg>
)

/**
 * TrendIndicator Component
 *
 * Displays a trend direction with appropriate slate color.
 * Uses green for up trends and red for down trends.
 *
 * Requirements: 5.2
 */
const TrendIndicator: React.FC<TrendProps> = ({ direction, value }) => (
  <div className={cn(trendVariants({ direction }))}>
    {direction === 'up' ? <TrendUpIcon /> : <TrendDownIcon />}
    <span>{value}</span>
  </div>
)

/**
 * StatCard Component
 *
 * A self-contained dashboard component that displays a single metric
 * with consistent glass card styling and centered content.
 *
 * Features:
 * - Glass card styling with centered content (Requirement 5.3)
 * - Label, value, and icon props (Requirement 5.1)
 * - Optional trend indicator with slate colors (Requirement 5.2)
 * - Identical styling across all portals (Requirement 5.4)
 *
 * @example
 * ```tsx
 * // Basic stat card
 * <StatCard label="Total Users" value={1234} />
 *
 * // Stat card with icon
 * <StatCard
 *   label="Revenue"
 *   value="$12,345"
 *   icon={<DollarIcon />}
 * />
 *
 * // Stat card with trend
 * <StatCard
 *   label="Growth"
 *   value="23%"
 *   trend={{ direction: "up", value: "+5%" }}
 * />
 *
 * // Full featured stat card
 * <StatCard
 *   label="Active Students"
 *   value={456}
 *   icon={<UsersIcon />}
 *   trend={{ direction: "up", value: "+12%" }}
 *   size="lg"
 * />
 * ```
 */
const StatCard = React.forwardRef<HTMLDivElement, StatCardProps>(
  ({ className, label, value, icon, trend, size, hover, ...props }, ref) => (
    <div ref={ref} className={cn(statCardVariants({ size, hover, className }))} {...props}>
      {/* Optional Icon - Requirement 5.1 */}
      {icon && <div className="text-muted-foreground mb-3">{icon}</div>}

      {/* Label - Requirement 5.1 */}
      <p className="text-sm text-muted-foreground font-medium">{label}</p>

      {/* Value - Requirement 5.1 */}
      <p className="text-3xl sm:text-4xl font-bold mt-1 text-foreground">{value}</p>

      {/* Optional Trend Indicator - Requirement 5.2 */}
      {trend && <TrendIndicator {...trend} />}
    </div>
  )
)
StatCard.displayName = 'StatCard'

export { StatCard, TrendIndicator, TrendUpIcon, TrendDownIcon }

/**
 * Type exports for external use
 */
export type StatCardSize = NonNullable<VariantProps<typeof statCardVariants>['size']>
export type StatCardHover = NonNullable<VariantProps<typeof statCardVariants>['hover']>
export type TrendDirection = TrendProps['direction']
