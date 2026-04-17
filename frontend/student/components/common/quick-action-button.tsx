import * as React from 'react';
import Link from 'next/link';
import { cn } from '@/lib/utils';

interface QuickActionButtonProps {
  icon: React.ReactNode;
  label: string;
  onClick?: () => void;
  href?: string;
  className?: string;
}

export const QuickActionButton = React.forwardRef<
  HTMLButtonElement | HTMLAnchorElement,
  QuickActionButtonProps
>(({ icon, label, onClick, href, className }, ref) => {
  const content = (
    <>
      <div
        className={cn(
          'w-12 h-12 rounded-xl flex items-center justify-center text-slate-600 dark:text-slate-300 mb-3 bg-slate-100 dark:bg-slate-800'
        )}
        aria-hidden="true"
      >
        {icon}
      </div>
      <span className="text-sm font-medium tracking-wide text-foreground text-center">{label}</span>
    </>
  );

  const baseClasses = cn(
    'flex flex-col items-center justify-center p-4 rounded-2xl',
    'border border-slate-200 dark:border-slate-700',
    'hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer',
    'focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2',
    'transition-all duration-300',
    className
  );

  if (href) {
    return (
      <Link
        href={href}
        className={baseClasses}
        aria-label={label}
        ref={ref as React.Ref<HTMLAnchorElement>}
      >
        {content}
      </Link>
    );
  }

  return (
    <button
      type="button"
      onClick={onClick}
      className={baseClasses}
      aria-label={label}
      ref={ref as React.Ref<HTMLButtonElement>}
    >
      {content}
    </button>
  );
});

QuickActionButton.displayName = 'QuickActionButton';
