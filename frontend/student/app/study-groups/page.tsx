'use client';

import { Users } from 'lucide-react';

export default function StudyGroupsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">Study Groups</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">Collaborate with your peers</p>
      </div>

      <div className="rounded-xl border border-[var(--color-border)] bg-white p-12 text-center shadow-sm">
        <Users className="mx-auto h-12 w-12 text-[var(--color-text-muted)]" />
        <h3 className="mt-4 text-base font-semibold text-[var(--color-text)]">
          Study Groups Coming Soon
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Create and join study groups with classmates for collaborative learning.
        </p>
      </div>
    </div>
  );
}
