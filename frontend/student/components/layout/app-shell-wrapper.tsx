'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { LogOut, User } from 'lucide-react';
import { AppShell } from '../../../shared/components/app-shell/app-shell';
import { TabNav, type TabItem } from '../../../shared/components/app-shell/tab-nav';
import type { NowBarChip } from '../../../shared/components/app-shell/now-bar';
import { auth } from '../../../shared/lib/api';
import { useStudentAssignments, useNextLecture } from '@/lib/api/hooks';
import { useStudentProfile, clearStudentProfile } from '@/lib/api/profile';

const STUDENT_TABS: TabItem[] = [
  { label: 'Today', href: '/today' },
  { label: 'Courses', href: '/courses' },
  { label: 'Assignments', href: '/assignments' },
  { label: 'Grades', href: '/grades' },
  { label: 'Plan', href: '/plan' },
  // TODO(wave-4): wire `count` to `/api/discussions/inbox` unread count
  //   so the Inbox tab shows a live badge.
  { label: 'Inbox', href: '/inbox' },
  { label: 'People', href: '/people' },
];

function hoursUntilIso(iso: string): number {
  return Math.round((new Date(iso).getTime() - Date.now()) / 3_600_000);
}

export function AppShellWrapper({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { data: profile } = useStudentProfile();
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  const openAssignments = useStudentAssignments({ status: 'open' });
  const nextLecture = useNextLecture();
  // NowBar OH chip was calling useOfficeHoursSlots() with no args from every
  // student page, but scheduling-service requires `instructor_id`. Result: 400
  // on 15 pages. Layout has no instructor context; the cosmetic chip isn't
  // worth a per-student ListSlots RPC. /office-hours page has its own call.

  const nowChips: NowBarChip[] = [];
  const firstAssignment = openAssignments.data?.[0];
  if (firstAssignment) {
    const h = hoursUntilIso(firstAssignment.dueDate);
    nowChips.push({
      id: 'ps',
      label: `${firstAssignment.title} · ${h <= 0 ? 'overdue' : `due in ${h}h`}`,
      tone: h <= 6 ? 'amber' : undefined,
      href: `/assignments/${firstAssignment.id}`,
    });
  }
  if (nextLecture.data) {
    nowChips.push({
      id: 'live',
      label: `Live: ${nextLecture.data.title}`,
      href: nextLecture.data.joinUrl ?? `/courses/${nextLecture.data.courseId}`,
    });
  }
  // OH-open chip removed — see comment above about layout-level slot queries.

  const initials = (() => {
    if (profile?.firstName || profile?.lastName) {
      const f = profile.firstName?.charAt(0) ?? '';
      const l = profile.lastName?.charAt(0) ?? '';
      const combined = `${f}${l}`;
      if (combined.trim()) return combined.toUpperCase();
    }
    if (profile?.email) return profile.email.charAt(0).toUpperCase();
    return 'ST';
  })();

  const handleLogout = async () => {
    try {
      await auth.logout();
    } finally {
      clearStudentProfile();
      if (typeof document !== 'undefined') {
        document.cookie = 'slate_token=; path=/; max-age=0';
      }
      router.push('/login');
    }
  };

  return (
    <AppShell
      brand={
        <Link
          href="/today"
          className="flex items-center gap-2 font-display text-lg tracking-tight text-forest-800"
        >
          <span
            aria-hidden="true"
            className="flex h-7 w-7 items-center justify-center rounded-md bg-gradient-to-br from-forest-500 to-forest-800 text-xs font-bold text-white"
          >
            S
          </span>
          <span>Slate</span>
        </Link>
      }
      centerNav={<TabNav tabs={STUDENT_TABS} />}
      right={
        <>
          <button
            type="button"
            aria-label="Open command palette"
            onClick={() => {
              window.dispatchEvent(
                new KeyboardEvent('keydown', { key: 'k', metaKey: true, ctrlKey: false })
              );
            }}
            className="hidden rounded-full px-2.5 py-1 text-xs font-medium text-warm-700 ring-1 ring-warm-200 hover:bg-warm-50 sm:inline-flex"
          >
            ⌘K
          </button>
          <div className="relative">
            <button
              type="button"
              onClick={() => setUserMenuOpen((v) => !v)}
              aria-label="User menu"
              aria-expanded={userMenuOpen}
              className="flex h-8 w-8 items-center justify-center rounded-full bg-forest-100 text-xs font-medium text-forest-700 ring-2 ring-forest-200"
            >
              {initials}
            </button>
            {userMenuOpen ? (
              <div
                role="menu"
                className="absolute right-0 z-50 mt-2 w-48 rounded-xl border border-warm-200 bg-white py-1 shadow-md"
              >
                <div className="border-b border-warm-200 px-4 py-2 text-sm">
                  <p className="font-medium text-warm-900">
                    {profile && (profile.firstName || profile.lastName)
                      ? `${profile.firstName} ${profile.lastName}`.trim()
                      : 'Student'}
                  </p>
                  <p className="text-xs text-warm-700">{profile?.email ?? ''}</p>
                </div>
                <Link
                  href="/profile"
                  role="menuitem"
                  onClick={() => setUserMenuOpen(false)}
                  className="flex items-center gap-2 px-4 py-2 text-sm text-warm-900 hover:bg-forest-50"
                >
                  <User className="h-3.5 w-3.5" />
                  Profile
                </Link>
                <button
                  type="button"
                  role="menuitem"
                  onClick={handleLogout}
                  className="flex w-full items-center gap-2 px-4 py-2 text-sm text-warm-900 hover:bg-forest-50"
                >
                  <LogOut className="h-3.5 w-3.5" />
                  Sign out
                </button>
              </div>
            ) : null}
          </div>
        </>
      }
      nowChips={nowChips}
      nowLabel="Today"
    >
      {children}
    </AppShell>
  );
}
