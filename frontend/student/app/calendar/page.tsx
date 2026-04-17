'use client';

import { Calendar } from 'lucide-react';

export default function CalendarPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">Calendar</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          View your schedule and upcoming events
        </p>
      </div>

      <div className="rounded-xl border border-[var(--color-border)] bg-white p-12 text-center shadow-sm">
        <Calendar className="mx-auto h-12 w-12 text-[var(--color-text-muted)]" />
        <h3 className="mt-4 text-base font-semibold text-[var(--color-text)]">
          Calendar Coming Soon
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Your class schedule, assignment deadlines, and events will appear here.
        </p>
      </div>
    </div>
  );
}
