/**
 * Slate Glass Design System - Widget Component
 *
 * A standardized widget component for building consistent dashboard layouts
 * across all portals. Uses glass card styling with header and content sections.
 *
 * Features:
 * - Glass card container with consistent styling
 * - Header section with title and optional icon
 * - Content section for children
 * - Responsive padding for mobile
 *
 * Requirements: 4.1, 4.2, 4.3
 */

import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../utils'

/**
 * Widget variant styles using class-variance-authority
 *
 * Base styles include:
 * - Glassmorphism with backdrop blur
 * - Slate-colored borders
 * - Rounded corners using design system radius
 *
 * Requirements: 4.1, 4.2, 4.3
 */
export const widgetVariants = cva(
  // Base glassmorphism styling
  [
    'rounded-xl',
    'bg-card/70 dark:bg-card/50',
    'backdrop-blur-[12px]',
    'border border-card-border',
    'text-card-foreground',
    'shadow-lg',
    'transition-colors duration-200',
    'overflow-hidden',
  ],
  {
    variants: {
      /**
       * Size variants for different widget sizes
       */
      size: {
        default: '',
        sm: '',
        lg: '',
        full: 'w-full',
      },
    },
    defaultVariants: {
      size: 'default',
    },
  }
)

/**
 * Widget header variant styles
 */
export const widgetHeaderVariants = cva(['flex items-center gap-3', 'border-b border-border/50'], {
  variants: {
    /**
     * Padding variants for responsive design
     */
    padding: {
      default: 'p-4 sm:p-6',
      sm: 'p-3 sm:p-4',
      lg: 'p-6 sm:p-8',
    },
  },
  defaultVariants: {
    padding: 'default',
  },
})

/**
 * Widget content variant styles
 */
export const widgetContentVariants = cva([], {
  variants: {
    /**
     * Padding variants for responsive design
     */
    padding: {
      default: 'p-4 sm:p-6',
      sm: 'p-3 sm:p-4',
      lg: 'p-6 sm:p-8',
      none: 'p-0',
    },
  },
  defaultVariants: {
    padding: 'default',
  },
})

/**
 * Widget component props
 *
 * Requirements: 4.1 - Widget SHALL accept title, icon, and children props
 */
export interface WidgetProps
  extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof widgetVariants> {
  /**
   * The title displayed in the widget header
   * Required for accessibility and consistent layout
   */
  title: string

  /**
   * Optional icon displayed before the title
   * Should be a React node (e.g., an icon component)
   */
  icon?: React.ReactNode

  /**
   * Content to render inside the widget
   */
  children: React.ReactNode

  /**
   * Padding size for header and content sections
   * @default "default"
   */
  padding?: 'default' | 'sm' | 'lg' | 'none'

  /**
   * Optional subtitle or description below the title
   */
  subtitle?: string

  /**
   * Optional action element (e.g., button) in the header
   */
  headerAction?: React.ReactNode
}

/**
 * Widget Component
 *
 * A self-contained dashboard component that displays specific information
 * with consistent glass card styling, header, and content sections.
 *
 * Features:
 * - Glass card container with consistent styling (Requirement 4.2)
 * - Header section with title and optional icon (Requirement 4.1, 4.3)
 * - Content section for children (Requirement 4.1)
 * - Responsive padding for mobile (Requirement 13.2, 13.3)
 * - Identical styling across all portals (Requirement 4.4)
 *
 * @example
 * ```tsx
 * // Basic widget
 * <Widget title="Recent Activity">
 *   <p>Activity content here</p>
 * </Widget>
 *
 * // Widget with icon
 * <Widget title="Progress" icon={<ChartIcon />}>
 *   <Progress value={75} />
 * </Widget>
 *
 * // Widget with header action
 * <Widget
 *   title="Tasks"
 *   icon={<TaskIcon />}
 *   headerAction={<Button size="sm">View All</Button>}
 * >
 *   <TaskList />
 * </Widget>
 * ```
 */
const Widget = React.forwardRef<HTMLDivElement, WidgetProps>(
  (
    {
      className,
      title,
      icon,
      children,
      size,
      padding = 'default',
      subtitle,
      headerAction,
      ...props
    },
    ref
  ) => (
    <div ref={ref} className={cn(widgetVariants({ size, className }))} {...props}>
      {/* Header Section - Requirement 4.3 */}
      <div
        className={cn(widgetHeaderVariants({ padding: padding === 'none' ? 'default' : padding }))}
      >
        <div className="flex items-center gap-3 flex-1 min-w-0">
          {/* Optional Icon - Requirement 4.1 */}
          {icon && <div className="flex-shrink-0 text-muted-foreground">{icon}</div>}

          {/* Title and Subtitle */}
          <div className="flex-1 min-w-0">
            <h3 className="text-lg font-semibold leading-none tracking-tight truncate">{title}</h3>
            {subtitle && <p className="text-sm text-muted-foreground mt-1 truncate">{subtitle}</p>}
          </div>
        </div>

        {/* Optional Header Action */}
        {headerAction && <div className="flex-shrink-0">{headerAction}</div>}
      </div>

      {/* Content Section - Requirement 4.1 */}
      <div className={cn(widgetContentVariants({ padding }))}>{children}</div>
    </div>
  )
)
Widget.displayName = 'Widget'

/**
 * WidgetHeader component for custom header layouts
 * Use this when you need more control over the header structure
 */
export interface WidgetHeaderProps
  extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof widgetHeaderVariants> {}

const WidgetHeader = React.forwardRef<HTMLDivElement, WidgetHeaderProps>(
  ({ className, padding, ...props }, ref) => (
    <div ref={ref} className={cn(widgetHeaderVariants({ padding, className }))} {...props} />
  )
)
WidgetHeader.displayName = 'WidgetHeader'

/**
 * WidgetContent component for custom content layouts
 * Use this when you need more control over the content structure
 */
export interface WidgetContentProps
  extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof widgetContentVariants> {}

const WidgetContent = React.forwardRef<HTMLDivElement, WidgetContentProps>(
  ({ className, padding, ...props }, ref) => (
    <div ref={ref} className={cn(widgetContentVariants({ padding, className }))} {...props} />
  )
)
WidgetContent.displayName = 'WidgetContent'

/**
 * WidgetTitle component for consistent title styling
 */
export interface WidgetTitleProps extends React.HTMLAttributes<HTMLHeadingElement> {}

const WidgetTitle = React.forwardRef<HTMLHeadingElement, WidgetTitleProps>(
  ({ className, ...props }, ref) => (
    <h3
      ref={ref}
      className={cn('text-lg font-semibold leading-none tracking-tight', className)}
      {...props}
    />
  )
)
WidgetTitle.displayName = 'WidgetTitle'

export { Widget, WidgetHeader, WidgetContent, WidgetTitle }

/**
 * Type exports for external use
 */
export type WidgetSize = NonNullable<VariantProps<typeof widgetVariants>['size']>
export type WidgetPadding = NonNullable<VariantProps<typeof widgetContentVariants>['padding']>
