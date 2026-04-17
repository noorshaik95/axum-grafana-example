'use client';

import { Megaphone, Loader2, AlertCircle, AlertTriangle, Info } from 'lucide-react';
import { useAnnouncements } from '@/lib/api/hooks';
import { formatRelativeTime } from '@/lib/utils';

const priorityStyles: Record<string, { icon: typeof Info; border: string; bg: string }> = {
  urgent: { icon: AlertTriangle, border: 'border-l-red-500', bg: 'bg-red-50' },
  high: { icon: AlertCircle, border: 'border-l-amber-500', bg: 'bg-amber-50' },
  normal: { icon: Megaphone, border: 'border-l-indigo-500', bg: 'bg-white' },
  low: { icon: Info, border: 'border-l-gray-300', bg: 'bg-white' },
};

export default function AnnouncementsPage() {
  const { data: announcements, isLoading, isError } = useAnnouncements();

  const list = Array.isArray(announcements) ? announcements : [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">Announcements</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Stay updated with the latest announcements from your courses
        </p>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="h-8 w-8 animate-spin text-[var(--color-text-muted)]" />
        </div>
      ) : isError ? (
        <div className="flex items-center justify-center gap-2 py-16 text-[var(--color-error)]">
          <AlertCircle className="h-5 w-5" />
          <span className="text-sm">Failed to load announcements</span>
        </div>
      ) : list.length === 0 ? (
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-16 text-center shadow-sm">
          <Megaphone className="mx-auto h-10 w-10 text-[var(--color-text-muted)]" />
          <p className="mt-3 text-sm text-[var(--color-text-muted)]">No announcements yet.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {list.map((announcement) => {
            const style = priorityStyles[announcement.priority] ?? priorityStyles.normal;
            const Icon = style.icon;
            const isUnread = !announcement.readAt;

            return (
              <div
                key={announcement.id}
                className={`rounded-xl border border-[var(--color-border)] border-l-4 ${style.border} ${style.bg} p-5 shadow-sm`}
              >
                <div className="flex items-start gap-3">
                  <Icon className="h-5 w-5 mt-0.5 text-[var(--color-text-muted)] shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="text-sm font-semibold text-[var(--color-text)]">
                        {announcement.title}
                      </h3>
                      {isUnread && <span className="h-2 w-2 rounded-full bg-indigo-600" />}
                    </div>
                    {announcement.courseTitle && (
                      <p className="text-xs text-indigo-600 font-medium mt-0.5">
                        {announcement.courseTitle}
                      </p>
                    )}
                    <p className="mt-2 text-sm text-[var(--color-text)] whitespace-pre-wrap">
                      {announcement.content}
                    </p>
                    <div className="mt-3 flex items-center gap-3 text-xs text-[var(--color-text-muted)]">
                      <span>{announcement.authorName}</span>
                      <span>{formatRelativeTime(announcement.createdAt)}</span>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
