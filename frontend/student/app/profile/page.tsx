'use client';

import { useState } from 'react';
import { User, Mail, Calendar, Loader2, Bell, Lock } from 'lucide-react';
import { useProfile } from '../../../shared/lib/api/hooks';
import { ActivityCalendar } from '@/components/progress/ActivityCalendar';

export default function ProfilePage() {
  const { data: profile, isLoading } = useProfile();
  const [notifications, setNotifications] = useState({
    email: true,
    push: true,
    assignments: true,
    grades: true,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--color-text-muted)]" />
      </div>
    );
  }

  const initials = profile ? `${profile.firstName.charAt(0)}${profile.lastName.charAt(0)}` : 'ST';

  return (
    <div className="space-y-6 max-w-3xl">
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">Profile</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Your personal information and settings
        </p>
      </div>

      {/* Profile card */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <div className="flex items-start gap-6">
          <div className="flex h-20 w-20 shrink-0 items-center justify-center rounded-full bg-indigo-100 text-2xl font-bold text-indigo-600">
            {initials}
          </div>
          <div className="flex-1">
            <h2 className="text-xl font-bold text-[var(--color-text)]">
              {profile?.firstName} {profile?.lastName}
            </h2>
            <div className="mt-3 space-y-2">
              <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                <Mail className="h-4 w-4" />
                <span>{profile?.email}</span>
              </div>
              <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                <User className="h-4 w-4" />
                <span className="capitalize">
                  {profile?.roles?.map((r) => r.name).join(', ') ?? 'Student'}
                </span>
              </div>
              <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                <Calendar className="h-4 w-4" />
                <span>
                  Joined{' '}
                  {profile?.createdAt
                    ? new Date(profile.createdAt).toLocaleDateString('en-US', {
                        month: 'long',
                        year: 'numeric',
                      })
                    : '--'}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Account info */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <h3 className="text-base font-semibold text-[var(--color-text)] mb-4">Account Details</h3>
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
              First Name
            </p>
            <p className="mt-1 text-sm text-[var(--color-text)]">{profile?.firstName ?? '--'}</p>
          </div>
          <div>
            <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
              Last Name
            </p>
            <p className="mt-1 text-sm text-[var(--color-text)]">{profile?.lastName ?? '--'}</p>
          </div>
          <div>
            <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
              Email
            </p>
            <p className="mt-1 text-sm text-[var(--color-text)]">{profile?.email ?? '--'}</p>
          </div>
          <div>
            <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
              Auth Method
            </p>
            <p className="mt-1 text-sm text-[var(--color-text)] capitalize">
              {profile?.authMethod ?? '--'}
            </p>
          </div>
        </div>
      </div>

      {/* Activity Calendar */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <ActivityCalendar />
      </div>

      {/* Notifications */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm space-y-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
            <Bell className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-base font-semibold text-[var(--color-text)]">Notifications</h2>
            <p className="text-sm text-[var(--color-text-muted)]">
              Choose what notifications you receive
            </p>
          </div>
        </div>
        <div className="space-y-2 pt-2">
          {[
            {
              key: 'email' as const,
              label: 'Email Notifications',
              desc: 'Receive updates via email',
            },
            { key: 'push' as const, label: 'Push Notifications', desc: 'Receive instant alerts' },
            {
              key: 'assignments' as const,
              label: 'Assignments',
              desc: 'New assignments and due dates',
            },
            { key: 'grades' as const, label: 'Grades', desc: 'When your work is graded' },
          ].map(({ key, label, desc }) => (
            <div
              key={key}
              className="flex items-center justify-between rounded-lg border border-[var(--color-border)] p-4"
            >
              <div>
                <p className="text-sm font-medium text-[var(--color-text)]">{label}</p>
                <p className="text-xs text-[var(--color-text-muted)]">{desc}</p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={notifications[key]}
                  onChange={(e) => setNotifications({ ...notifications, [key]: e.target.checked })}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-indigo-200 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-indigo-600" />
              </label>
            </div>
          ))}
        </div>
      </div>

      {/* Security */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm space-y-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
            <Lock className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-base font-semibold text-[var(--color-text)]">Security</h2>
            <p className="text-sm text-[var(--color-text-muted)]">Manage your account security</p>
          </div>
        </div>
        <div className="space-y-2 pt-2">
          <button className="w-full rounded-lg border border-[var(--color-border)] px-4 py-2.5 text-sm font-medium text-[var(--color-text)] hover:bg-[var(--color-bg-muted)] transition-colors text-left">
            Change Password
          </button>
          <button className="w-full rounded-lg border border-[var(--color-border)] px-4 py-2.5 text-sm font-medium text-[var(--color-text)] hover:bg-[var(--color-bg-muted)] transition-colors text-left">
            Enable Two-Factor Authentication
          </button>
        </div>
      </div>
    </div>
  );
}
