/**
 * Slate Glass Design System - Button Component (Student Portal)
 *
 * A standardized button component with consistent styling.
 * Uses the slate color palette with blue-500 accent for primary actions.
 *
 * Variants:
 * - primary: Blue-500 background, white text (only colored element)
 * - secondary: Transparent background with slate border
 * - ghost: Transparent background with slate text
 * - destructive: Red background for dangerous actions
 * - outline: Transparent with visible border
 * - link: Text-only with underline on hover
 *
 * Requirements: 8.1, 8.2, 8.3, 8.4, 8.5
 */

import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

/**
 * Button variant styles using class-variance-authority
 *
 * Base styles include:
 * - Inline flex layout with centered content
 * - Rounded corners using design system radius
 * - Focus-visible ring using slate-500 (Requirement 14.2)
 * - Disabled state handling
 * - Transition for smooth hover effects (no glow)
 *
 * Requirements: 8.1, 8.2, 8.3, 8.4, 8.5
 */
const buttonVariants = cva(
  // Base styles - consistent across all variants
  [
    'inline-flex items-center justify-center gap-2',
    'whitespace-nowrap rounded-lg text-sm font-medium',
    'transition-colors duration-200',
    // Focus styles using slate-500 ring (Requirement 14.2)
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500 focus-visible:ring-offset-2',
    // Disabled state
    'disabled:pointer-events-none disabled:opacity-50',
    // SVG icon handling
    '[&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0',
  ],
  {
    variants: {
      /**
       * Button variants following Slate Glass design system
       * No aurora, glow, or neon effects (Requirement 8.5)
       */
      variant: {
        /**
         * Primary variant (Requirement 8.1)
         * Blue-500 background with white text - the only colored interactive element
         * Hover: Blue-600 background (subtle shift, no glow)
         */
        primary: ['bg-blue-500 text-white', 'hover:bg-blue-600', 'shadow-sm'],

        /**
         * Default variant - same as primary for backwards compatibility
         */
        default: ['bg-blue-500 text-white', 'hover:bg-blue-600', 'shadow-sm'],

        /**
         * Secondary variant (Requirement 8.2)
         * Transparent background with slate border
         * Hover: Subtle slate background
         */
        secondary: [
          'bg-transparent border border-slate-300 dark:border-slate-600',
          'text-slate-900 dark:text-slate-100',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
        ],

        /**
         * Ghost variant (Requirement 8.3)
         * Transparent background with slate text
         * Hover: Subtle slate background
         */
        ghost: [
          'bg-transparent text-slate-900 dark:text-slate-100',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
        ],

        /**
         * Destructive variant
         * Red background for dangerous actions
         * Hover: Darker red (no glow)
         */
        destructive: ['bg-red-500 text-white', 'hover:bg-red-600', 'shadow-sm'],

        /**
         * Outline variant
         * Transparent with visible slate border
         * Hover: Subtle background with border color change
         */
        outline: [
          'bg-transparent border border-slate-300 dark:border-slate-600',
          'text-slate-900 dark:text-slate-100',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
          'hover:border-slate-400 dark:hover:border-slate-500',
        ],

        /**
         * Link variant
         * Text-only with underline on hover
         * Uses primary color (blue-500)
         */
        link: ['bg-transparent text-blue-500', 'underline-offset-4 hover:underline', 'h-auto p-0'],

        /**
         * Glow variant (deprecated - maps to primary)
         * Kept for backwards compatibility during migration
         * Will be removed after full migration is complete
         * @deprecated Use 'primary' variant instead
         */
        glow: ['bg-blue-500 text-white', 'hover:bg-blue-600', 'shadow-sm'],

        /**
         * Aurora variant (deprecated - maps to primary)
         * Kept for backwards compatibility during migration
         * Will be removed after full migration is complete
         * @deprecated Use 'primary' variant instead
         */
        aurora: ['bg-blue-500 text-white', 'hover:bg-blue-600', 'shadow-sm'],
      },

      /**
       * Size variants
       */
      size: {
        default: 'h-10 px-4 py-2',
        sm: 'h-9 rounded-md px-3 text-xs',
        lg: 'h-11 rounded-lg px-8 text-base',
        icon: 'h-10 w-10 p-0',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> {
  /**
   * When true, the button will render as a Slot component,
   * allowing the button styles to be applied to a child element
   */
  asChild?: boolean;
}

/**
 * Button Component
 *
 * A versatile button component that supports multiple variants and sizes.
 * Follows the Slate Glass design system with:
 * - Blue-500 accent for primary actions (Requirement 8.1)
 * - Slate-based styling for secondary/ghost variants (Requirements 8.2, 8.3)
 * - No glow or aurora effects (Requirement 8.5)
 * - Accessible focus indicators using slate-500 ring (Requirement 14.2)
 *
 * @example
 * ```tsx
 * // Primary button (blue accent)
 * <Button variant="primary">Save Changes</Button>
 *
 * // Secondary button (slate border)
 * <Button variant="secondary">Cancel</Button>
 *
 * // Ghost button (transparent)
 * <Button variant="ghost">Learn More</Button>
 *
 * // Destructive button (red)
 * <Button variant="destructive">Delete</Button>
 * ```
 */
const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button';
    return (
      <Comp className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />
    );
  }
);
Button.displayName = 'Button';

export { Button, buttonVariants };
