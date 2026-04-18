'use client';

import Link from 'next/link';
import { Loader2, AlertCircle, Send, Inbox } from 'lucide-react';
import { useSentMessages } from '@/lib/api/hooks';
import { formatRelativeTime } from '@/lib/utils';

export default function SentMessagesPage() {
  const { data: threads, isLoading, isError } = useSentMessages();

  const threadList = Array.isArray(threads) ? threads : [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-[var(--color-text)]">Messages</h1>
          <p className="mt-1 text-sm text-[var(--color-text-muted)]">Sent messages</p>
        </div>
        <div className="flex items-center gap-2">
          <Link
            href="/messages"
            className="rounded-lg border border-[var(--color-border)] bg-white px-3 py-1.5 text-xs font-medium text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)]"
          >
            <Inbox className="h-3.5 w-3.5 inline mr-1" />
            Inbox
          </Link>
          <Link
            href="/messages/sent"
            className="rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white"
          >
            <Send className="h-3.5 w-3.5 inline mr-1" />
            Sent
          </Link>
        </div>
      </div>

      <div className="rounded-xl border border-[var(--color-border)] bg-white shadow-sm">
        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="h-8 w-8 animate-spin text-[var(--color-text-muted)]" />
          </div>
        ) : isError ? (
          <div className="flex items-center justify-center gap-2 py-16 text-[var(--color-error)]">
            <AlertCircle className="h-5 w-5" />
            <span className="text-sm">Failed to load messages</span>
          </div>
        ) : threadList.length === 0 ? (
          <div className="py-16 text-center">
            <Send className="mx-auto h-10 w-10 text-[var(--color-text-muted)]" />
            <p className="mt-3 text-sm text-[var(--color-text-muted)]">No sent messages.</p>
          </div>
        ) : (
          <div className="divide-y divide-[var(--color-border)]">
            {threadList.map((thread) => (
              <Link
                key={thread.id}
                href={`/messages/${thread.id}`}
                className="flex items-center gap-4 p-4 hover:bg-[var(--color-bg-muted)]/50 transition-colors"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gray-100 text-sm font-bold text-gray-600">
                  <Send className="h-4 w-4" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-[var(--color-text)] truncate">
                    {thread.subject}
                  </p>
                  <p className="text-xs text-[var(--color-text-muted)] truncate mt-0.5">
                    To: {thread.participants.map((p) => p.name).join(', ')}
                  </p>
                </div>
                <div className="text-xs text-[var(--color-text-muted)] shrink-0">
                  {formatRelativeTime(thread.lastMessageAt)}
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
