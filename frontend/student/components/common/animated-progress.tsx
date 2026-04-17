import * as React from 'react';
import { cn } from '@/lib/utils';
import { GradientType } from './gradient-card';

interface AnimatedProgressProps {
  value: number;
  gradient?: GradientType;
  showLabel?: boolean;
  size?: 'sm' | 'md' | 'lg';
  animated?: boolean;
  className?: string;
}

const sizeClasses = {
  sm: 'h-1.5',
  md: 'h-2.5',
  lg: 'h-3',
};

export const AnimatedProgress = React.forwardRef<HTMLDivElement, AnimatedProgressProps>(
  ({ value, showLabel = false, size = 'md', animated = true, className }, ref) => {
    const clampedValue = Math.min(Math.max(value, 0), 100);

    return (
      <div ref={ref} className={cn('w-full', className)}>
        <div className="flex items-center justify-between mb-1">
          {showLabel && (
            <span className="text-sm font-medium text-muted-foreground">{clampedValue}%</span>
          )}
        </div>
        <div
          className={cn(
            'w-full bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden',
            sizeClasses[size]
          )}
        >
          <div
            className={cn(
              'h-full rounded-full transition-all duration-500 ease-out bg-slate-600 dark:bg-slate-400',
              animated && 'shimmer-animation'
            )}
            style={{ width: `${clampedValue}%` }}
            role="progressbar"
            aria-valuenow={clampedValue}
            aria-valuemin={0}
            aria-valuemax={100}
          />
        </div>
      </div>
    );
  }
);

AnimatedProgress.displayName = 'AnimatedProgress';
