'use client';

import Link from 'next/link';
import { Video, Clock, Users, Loader2, AlertCircle, Calendar, Play } from 'lucide-react';
import { useVideoSessions } from '@/lib/api/hooks';
import { formatDateTime } from '@/lib/utils';

export default function VideoSessionsPage() {
  const { data: sessions, isLoading, isError } = useVideoSessions();

  const sessionList = Array.isArray(sessions) ? sessions : [];
  const activeSessions = sessionList.filter((s) => s.isActive);
  const upcomingSessions = sessionList.filter(
    (s) => !s.isActive && new Date(s.startTime) > new Date()
  );
  const pastSessions = sessionList.filter(
    (s) => !s.isActive && new Date(s.startTime) <= new Date()
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">Live Classes</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Join live classes and watch recordings
        </p>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="h-8 w-8 animate-spin text-[var(--color-text-muted)]" />
        </div>
      ) : isError ? (
        <div className="flex items-center justify-center gap-2 py-16 text-[var(--color-error)]">
          <AlertCircle className="h-5 w-5" />
          <span className="text-sm">Failed to load sessions</span>
        </div>
      ) : sessionList.length === 0 ? (
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-16 text-center shadow-sm">
          <Video className="mx-auto h-12 w-12 text-[var(--color-text-muted)]" />
          <h3 className="mt-3 text-base font-semibold text-[var(--color-text)]">No sessions</h3>
          <p className="mt-1 text-sm text-[var(--color-text-muted)]">
            No live classes are scheduled at this time.
          </p>
        </div>
      ) : (
        <>
          {activeSessions.length > 0 && (
            <div className="space-y-3">
              <h2 className="text-base font-semibold text-[var(--color-text)]">Live Now</h2>
              <div className="grid gap-4 sm:grid-cols-2">
                {activeSessions.map((session) => (
                  <Link
                    key={session.id}
                    href={`/video/${session.id}`}
                    className="group rounded-xl border-2 border-red-200 bg-white p-5 shadow-sm hover:shadow-md transition-shadow"
                  >
                    <div className="flex items-center gap-2 mb-2">
                      <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-700">
                        <span className="h-1.5 w-1.5 rounded-full bg-red-600 animate-pulse" />
                        LIVE
                      </span>
                      <span className="text-xs text-[var(--color-text-muted)]">
                        {session.courseTitle}
                      </span>
                    </div>
                    <p className="text-sm font-semibold text-[var(--color-text)] group-hover:text-indigo-600 transition-colors">
                      {session.title}
                    </p>
                    <p className="text-xs text-[var(--color-text-muted)] mt-1">
                      {session.hostName}
                    </p>
                    <div className="mt-3 flex items-center gap-3 text-xs text-[var(--color-text-muted)]">
                      <span className="flex items-center gap-1">
                        <Users className="h-3 w-3" />
                        {session.participantCount}/{session.maxParticipants}
                      </span>
                    </div>
                    <div className="mt-3">
                      <span className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white group-hover:bg-indigo-700 transition-colors">
                        <Play className="h-3 w-3" /> Join Now
                      </span>
                    </div>
                  </Link>
                ))}
              </div>
            </div>
          )}

          {upcomingSessions.length > 0 && (
            <div className="space-y-3">
              <h2 className="text-base font-semibold text-[var(--color-text)]">Upcoming</h2>
              <div className="grid gap-4 sm:grid-cols-2">
                {upcomingSessions.map((session) => (
                  <Link
                    key={session.id}
                    href={`/video/${session.id}`}
                    className="group rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm hover:shadow-md transition-shadow"
                  >
                    <span className="text-xs text-[var(--color-text-muted)]">
                      {session.courseTitle}
                    </span>
                    <p className="mt-1 text-sm font-semibold text-[var(--color-text)] group-hover:text-indigo-600 transition-colors">
                      {session.title}
                    </p>
                    <p className="text-xs text-[var(--color-text-muted)] mt-1">
                      {session.hostName}
                    </p>
                    <div className="mt-3 flex items-center gap-3 text-xs text-[var(--color-text-muted)]">
                      <span className="flex items-center gap-1">
                        <Calendar className="h-3 w-3" />
                        {formatDateTime(session.startTime)}
                      </span>
                      <span className="flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {session.duration} min
                      </span>
                    </div>
                  </Link>
                ))}
              </div>
            </div>
          )}

          {pastSessions.length > 0 && (
            <div className="space-y-3">
              <h2 className="text-base font-semibold text-[var(--color-text)]">Past Sessions</h2>
              <div className="rounded-xl border border-[var(--color-border)] bg-white shadow-sm divide-y divide-[var(--color-border)]">
                {pastSessions.map((session) => (
                  <div key={session.id} className="flex items-center justify-between p-4">
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-[var(--color-text)] truncate">
                        {session.title}
                      </p>
                      <p className="text-xs text-[var(--color-text-muted)]">
                        {session.courseTitle} -- {formatDateTime(session.startTime)}
                      </p>
                    </div>
                    {session.recordingUrl ? (
                      <a
                        href={session.recordingUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-xs font-medium text-indigo-600 hover:text-indigo-700 shrink-0 ml-3"
                      >
                        <Play className="h-3 w-3" /> Watch Recording
                      </a>
                    ) : (
                      <span className="text-xs text-[var(--color-text-muted)] shrink-0 ml-3">
                        No recording
                      </span>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
