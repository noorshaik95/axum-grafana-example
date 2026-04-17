import * as React from 'react';
import { cn } from '@/lib/utils';

interface StatCardProps {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  subtitle?: string;
  variant?: 'solid' | 'outline';
  className?: string;
}

export const StatCard = React.forwardRef<HTMLDivElement, StatCardProps>(
  ({ icon, label, value, subtitle, variant = 'outline', className }, ref) => {
    if (variant === 'solid') {
      return (
        <div
          ref={ref}
          className={cn('rounded-2xl p-6 bg-slate-700 dark:bg-slate-800 text-white', className)}
        >
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <p className="text-sm font-medium text-white/80 mb-1">{label}</p>
              <p className="text-3xl font-bold mb-1">{value}</p>
              {subtitle && <p className="text-sm text-white/70">{subtitle}</p>}
            </div>
            <div className="ml-4 p-3 bg-white/20 backdrop-blur-sm rounded-xl">{icon}</div>
          </div>
        </div>
      );
    }

    return (
      <div ref={ref} className={cn('glass-card rounded-2xl p-6', className)}>
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <p className="text-sm font-medium text-muted-foreground mb-1">{label}</p>
            <p className="text-3xl font-bold text-foreground mb-1">{value}</p>
            {subtitle && <p className="text-sm text-muted-foreground">{subtitle}</p>}
          </div>
          <div className="ml-4 p-3 rounded-xl bg-blue-500 text-white">{icon}</div>
        </div>
      </div>
    );
  }
);

StatCard.displayName = 'StatCard';
