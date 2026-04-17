'use client';

import { Trophy } from 'lucide-react';

export default function AchievementsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">Achievements</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Track your milestones and badges
        </p>
      </div>

      <div className="rounded-xl border border-[var(--color-border)] bg-white p-12 text-center shadow-sm">
        <Trophy className="mx-auto h-12 w-12 text-[var(--color-text-muted)]" />
        <h3 className="mt-4 text-base font-semibold text-[var(--color-text)]">
          Achievements Coming Soon
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Earn badges and track your academic milestones as you progress through courses.
        </p>
      </div>
    </div>
  );
}
