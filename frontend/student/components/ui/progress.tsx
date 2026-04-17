/**
 * Slate Glass Design System - Progress Component (Student Portal)
 *
 * A standardized progress indicator component with consistent styling.
 * Uses the slate color palette for track and fill colors.
 *
 * Features:
 * - Slate-colored track and fill
 * - Dark mode: slate-700 track, slate-400 fill
 * - Light mode: slate-200 track, slate-600 fill
 * - No neon glow or animated gradient effects
 * - Optional label and value display
 *
 * Requirements: 7.1, 7.2, 7.3, 7.4
 */

import * as React from 'react';
import * as ProgressPrimitive from '@radix-ui/react-progress';
import { cn } from '@/lib/utils';
import { cva, type VariantProps } from 'class-variance-authority';

/**
 * Progress track variant styles using class-variance-authority
 *
 * Base styles include:
 * - Slate-colored track background
 * - Rounded corners
 * - Overflow hidden for fill containment
 * - No glow or neon effects
 *
 * Requirements: 7.2, 7.3, 7.4
 */
const progressTrackVariants = cva(
  // Base track styling - slate colors, no glow
  [
    'relative w-full overflow-hidden rounded-full',
    // Light mode: slate-200 track (Requirement 7.4)
    'bg-slate-200',
    // Dark mode: slate-700 track (Requirement 7.3)
    'dark:bg-slate-700',
  ],
  {
    variants: {
      /**
       * Size variants for different use cases
       */
      size: {
        default: 'h-2',
        sm: 'h-1',
        md: 'h-3',
        lg: 'h-4',
      },
    },
    defaultVariants: {
      size: 'default',
    },
  }
);

/**
 * Progress fill variant styles using class-variance-authority
 *
 * Base styles include:
 * - Slate-colored fill
 * - Smooth transition for value changes
 * - No neon glow or animated gradient
 *
 * Requirements: 7.2, 7.3, 7.4
 */
const progressFillVariants = cva(
  // Base fill styling - slate colors, no glow
  [
    'h-full rounded-full',
    // Light mode: slate-600 fill (Requirement 7.4)
    'bg-slate-600',
    // Dark mode: slate-400 fill (Requirement 7.3)
    'dark:bg-slate-400',
    // Smooth transition for value changes (respects reduced motion)
    'transition-all duration-300 ease-out',
  ],
  {
    variants: {
      /**
       * Animation variants
       * No neon glow or shimmer effects (Requirement 7.2)
       */
      animated: {
        true: '',
        false: 'transition-none',
      },
    },
    defaultVariants: {
      animated: true,
    },
  }
);

export interface ProgressProps
  extends
    React.ComponentPropsWithoutRef<typeof ProgressPrimitive.Root>,
    VariantProps<typeof progressTrackVariants> {
  /**
   * Optional label displayed above the progress bar
   */
  label?: string;
  /**
   * Whether to show the current value as text
   */
  showValue?: boolean;
  /**
   * Whether to animate value changes
   */
  animated?: boolean;
}

/**
 * Progress Component
 *
 * A progress indicator with slate-themed styling.
 * Supports light and dark mode with appropriate track and fill colors.
 *
 * Features:
 * - Light mode: slate-200 track, slate-600 fill (Requirement 7.4)
 * - Dark mode: slate-700 track, slate-400 fill (Requirement 7.3)
 * - No neon glow or animated gradient (Requirement 7.2)
 * - Optional label and value display (Requirement 7.1)
 *
 * @example
 * ```tsx
 * // Basic progress bar
 * <Progress value={50} />
 *
 * // With label and value display
 * <Progress value={75} max={100} label="Course Progress" showValue />
 *
 * // Different sizes
 * <Progress value={30} size="sm" />
 * <Progress value={60} size="lg" />
 *
 * // Without animation
 * <Progress value={100} animated={false} />
 * ```
 */
const Progress = React.forwardRef<React.ElementRef<typeof ProgressPrimitive.Root>, ProgressProps>(
  (
    { className, value, max = 100, label, showValue = false, size, animated = true, ...props },
    ref
  ) => {
    // Calculate percentage, clamped between 0 and 100
    const percentage = Math.min(100, Math.max(0, ((value || 0) / max) * 100));

    return (
      <div className={cn('w-full', className)}>
        {/* Label and value row */}
        {(label || showValue) && (
          <div className="flex items-center justify-between mb-1.5">
            {label && (
              <span className="text-sm font-medium text-slate-900 dark:text-slate-50">{label}</span>
            )}
            {showValue && (
              <span className="text-sm text-slate-500 dark:text-slate-400">
                {Math.round(percentage)}%
              </span>
            )}
          </div>
        )}

        {/* Progress track */}
        <ProgressPrimitive.Root
          ref={ref}
          className={cn(progressTrackVariants({ size }))}
          value={value}
          max={max}
          {...props}
        >
          {/* Progress fill */}
          <ProgressPrimitive.Indicator
            className={cn(progressFillVariants({ animated }))}
            style={{ transform: `translateX(-${100 - percentage}%)` }}
          />
        </ProgressPrimitive.Root>
      </div>
    );
  }
);
Progress.displayName = ProgressPrimitive.Root.displayName;

export { Progress, progressTrackVariants, progressFillVariants };
