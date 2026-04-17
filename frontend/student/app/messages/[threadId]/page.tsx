'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Loader2, AlertCircle, Send } from 'lucide-react';
import { useMessageThread, useReplyToThread } from '@/lib/api/hooks';
import { formatDateTime } from '@/lib/utils';

export default function ThreadDetailPage() {
  const params = useParams();
  const threadId = params.threadId as string;
  const [replyContent, setReplyContent] = useState('');

  const { data, isLoading, isError } = useMessageThread(threadId);
  const replyMutation = useReplyToThread();

  const handleReply = (e: React.FormEvent) => {
    e.preventDefault();
    if (!replyContent.trim()) return;
    replyMutation.mutate(
      { threadId, content: replyContent },
      { onSuccess: () => setReplyContent('') }
    );
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--color-text-muted)]" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="flex flex-col items-center py-20 text-center">
        <AlertCircle className="h-12 w-12 text-[var(--color-error)] mb-3" />
        <h2 className="text-lg font-semibold text-[var(--color-text)]">Thread not found</h2>
        <Link
          href="/messages"
          className="mt-4 inline-flex items-center gap-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-700"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Messages
        </Link>
      </div>
    );
  }

  const { thread, messages } = data;

  return (
    <div className="max-w-3xl space-y-4">
      <Link
        href="/messages"
        className="inline-flex items-center gap-1.5 text-sm font-medium text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Messages
      </Link>

      {/* Thread header */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
        <h1 className="text-lg font-bold text-[var(--color-text)]">{thread.subject}</h1>
        {thread.courseTitle && (
          <p className="text-xs text-indigo-600 font-medium mt-1">{thread.courseTitle}</p>
        )}
        <div className="mt-2 flex flex-wrap gap-2">
          {thread.participants.map((p) => (
            <span
              key={p.id}
              className="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs text-[var(--color-text-muted)]"
            >
              {p.name} ({p.role})
            </span>
          ))}
        </div>
      </div>

      {/* Messages */}
      <div className="space-y-3">
        {messages.map((msg) => (
          <div
            key={msg.id}
            className="rounded-xl border border-[var(--color-border)] bg-white p-4 shadow-sm"
          >
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-indigo-100 text-xs font-bold text-indigo-600">
                  {msg.senderName.charAt(0)}
                </div>
                <div>
                  <p className="text-sm font-medium text-[var(--color-text)]">{msg.senderName}</p>
                  <p className="text-[10px] text-[var(--color-text-muted)]">{msg.senderRole}</p>
                </div>
              </div>
              <span className="text-xs text-[var(--color-text-muted)]">
                {formatDateTime(msg.createdAt)}
              </span>
            </div>
            <p className="text-sm text-[var(--color-text)] whitespace-pre-wrap">{msg.content}</p>
          </div>
        ))}
      </div>

      {/* Reply */}
      <form
        onSubmit={handleReply}
        className="rounded-xl border border-[var(--color-border)] bg-white p-4 shadow-sm"
      >
        <textarea
          value={replyContent}
          onChange={(e) => setReplyContent(e.target.value)}
          placeholder="Type your reply..."
          rows={3}
          className="w-full rounded-lg border border-[var(--color-border)] p-3 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none"
        />
        <div className="mt-2 flex justify-end">
          <button
            type="submit"
            disabled={!replyContent.trim() || replyMutation.isPending}
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 transition-colors"
          >
            {replyMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
            Reply
          </button>
        </div>
      </form>
    </div>
  );
}
