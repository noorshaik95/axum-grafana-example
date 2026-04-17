'use client';

import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Loader2, AlertCircle } from 'lucide-react';
import { useVideoSession } from '@/lib/api/hooks';
import { JoinSession } from '@/components/video/JoinSession';

export default function VideoSessionDetailPage() {
  const params = useParams();
  const router = useRouter();
  const sessionId = params.sessionId as string;

  const { data: session, isLoading, isError } = useVideoSession(sessionId);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--color-text-muted)]" />
      </div>
    );
  }

  if (isError || !session) {
    return (
      <div className="flex flex-col items-center py-20 text-center">
        <AlertCircle className="h-12 w-12 text-[var(--color-error)] mb-3" />
        <h2 className="text-lg font-semibold text-[var(--color-text)]">Session not found</h2>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          This session could not be loaded.
        </p>
        <Link
          href="/video"
          className="mt-4 inline-flex items-center gap-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-700"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Sessions
        </Link>
      </div>
    );
  }

  const handleJoin = () => {
    if (session.joinUrl) {
      window.open(session.joinUrl, '_blank', 'noopener,noreferrer');
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-4">
      <Link
        href="/video"
        className="inline-flex items-center gap-1.5 text-sm font-medium text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Sessions
      </Link>

      <JoinSession session={session} onJoin={handleJoin} />
    </div>
  );
}
