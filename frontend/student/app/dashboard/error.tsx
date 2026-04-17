'use client';

import { AlertCircle, RefreshCw } from 'lucide-react';

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <AlertCircle className="h-12 w-12 text-[var(--color-error)] mb-4" />
      <h2 className="text-lg font-semibold text-[var(--color-text)]">Something went wrong</h2>
      <p className="mt-1 text-sm text-[var(--color-text-muted)] max-w-md">
        {error.message || 'An unexpected error occurred while loading the dashboard.'}
      </p>
      <button
        onClick={reset}
        className="mt-4 inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors"
      >
        <RefreshCw className="h-4 w-4" />
        Try Again
      </button>
    </div>
  );
}
