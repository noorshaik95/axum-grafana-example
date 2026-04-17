/**
 * Slate Glass Design System - Card Component (Student Portal)
 *
 * A standardized card component with glassmorphism styling.
 * Uses the slate color palette with subtle hover effects (no glow).
 *
 * Features:
 * - Glass card styling with backdrop blur
 * - Light mode: white/70 background, slate-200 border
 * - Dark mode: slate-900/50 background, slate-700 border
 * - Subtle border change on hover (no neon glow)
 * - Responsive padding for mobile
 *
 * Requirements: 3.1, 3.2, 3.3, 3.4
 */

import * as React from 'react';
import { cn } from '@/lib/utils';
import { cva, type VariantProps } from 'class-variance-authority';

/**
 * Card variant styles using class-variance-authority
 *
 * Base styles include:
 * - Glassmorphism with backdrop blur
 * - Slate-colored borders
 * - Subtle hover state (border change only, no glow)
 * - Rounded corners using design system radius
 *
 * Requirements: 3.1, 3.2, 3.3, 3.4
 */
const cardVariants = cva(
  // Base glassmorphism styling
  [
    'rounded-xl',
    // Light mode: white/70 background (Requirement 3.2)
    'bg-white/70',
    // Dark mode: slate-900/50 background (Requirement 3.1)
    'dark:bg-slate-900/50',
    'backdrop-blur-[12px]',
    // Light mode: slate-200 border (Requirement 3.2)
    'border border-slate-200',
    // Dark mode: slate-700 border (Requirement 3.1)
    'dark:border-slate-700',
    'text-slate-900 dark:text-slate-50',
    'shadow-lg',
    'transition-colors duration-200',
  ],
  {
    variants: {
      /**
       * Hover behavior variants
       * No glow effects - only subtle border changes (Requirement 3.3, 3.4)
       */
      hover: {
        /**
         * Default hover: subtle border color change
         * Light mode: border becomes slightly more visible
         * Dark mode: border lightens slightly
         */
        default: ['hover:border-slate-300', 'dark:hover:border-slate-600'],
        /**
         * No hover effect
         */
        none: '',
        /**
         * Lift effect: subtle elevation change without glow
         */
        lift: [
          'hover:border-slate-300',
          'dark:hover:border-slate-600',
          'hover:-translate-y-1',
          'hover:shadow-xl',
          'transition-all',
        ],
      },
      /**
       * Padding variants for different use cases
       */
      padding: {
        default: '',
        none: 'p-0',
        sm: 'p-3',
        md: 'p-4 sm:p-6',
        lg: 'p-6 sm:p-8',
      },
    },
    defaultVariants: {
      hover: 'default',
      padding: 'default',
    },
  }
);

export interface CardProps
  extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof cardVariants> {}

/**
 * Card Component
 *
 * A glassmorphism card container with slate-themed styling.
 * Supports light and dark mode with appropriate backgrounds and borders.
 *
 * Features:
 * - Light mode: white/70 background, slate-200 border (Requirement 3.2)
 * - Dark mode: slate-900/50 background, slate-700 border (Requirement 3.1)
 * - Hover: subtle border change without glow (Requirement 3.3, 3.4)
 *
 * @example
 * ```tsx
 * <Card>
 *   <CardHeader>
 *     <CardTitle>Card Title</CardTitle>
 *     <CardDescription>Card description</CardDescription>
 *   </CardHeader>
 *   <CardContent>
 *     <p>Card content goes here</p>
 *   </CardContent>
 *   <CardFooter>
 *     <Button>Action</Button>
 *   </CardFooter>
 * </Card>
 * ```
 */
const Card = React.forwardRef<HTMLDivElement, CardProps>(
  ({ className, hover, padding, ...props }, ref) => (
    <div ref={ref} className={cn(cardVariants({ hover, padding, className }))} {...props} />
  )
);
Card.displayName = 'Card';

/**
 * CardHeader Component
 *
 * Container for card title and description with responsive padding.
 * Smaller padding on mobile (Requirement 13.2, 13.3)
 */
const CardHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn('flex flex-col space-y-1.5 p-4 sm:p-6', className)} {...props} />
  )
);
CardHeader.displayName = 'CardHeader';

/**
 * CardTitle Component
 *
 * Card heading with consistent typography.
 */
const CardTitle = React.forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...props }, ref) => (
    <h3
      ref={ref}
      className={cn('text-2xl font-semibold leading-none tracking-tight', className)}
      {...props}
    />
  )
);
CardTitle.displayName = 'CardTitle';

/**
 * CardDescription Component
 *
 * Secondary text for card with muted foreground color.
 */
const CardDescription = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement>
>(({ className, ...props }, ref) => (
  <p ref={ref} className={cn('text-sm text-slate-500 dark:text-slate-400', className)} {...props} />
));
CardDescription.displayName = 'CardDescription';

/**
 * CardContent Component
 *
 * Main content area with responsive padding.
 * Smaller padding on mobile (Requirement 13.2, 13.3)
 */
const CardContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn('p-4 sm:p-6 pt-0', className)} {...props} />
  )
);
CardContent.displayName = 'CardContent';

/**
 * CardFooter Component
 *
 * Footer area for actions with responsive padding and flex-wrap for mobile.
 * (Requirement 13.2, 13.3)
 */
const CardFooter = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn('flex flex-wrap items-center gap-2 p-4 sm:p-6 pt-0', className)}
      {...props}
    />
  )
);
CardFooter.displayName = 'CardFooter';

export { Card, CardHeader, CardFooter, CardTitle, CardDescription, CardContent, cardVariants };
