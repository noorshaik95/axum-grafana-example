'use client';

import { useState, useEffect } from 'react';
import { Video, VideoOff, Mic, MicOff, Clock, Users, ExternalLink, Loader2 } from 'lucide-react';
import type { VideoSession } from '@/lib/api/video';

interface JoinSessionProps {
  session: VideoSession;
  onJoin: () => void;
}

function CountdownTimer({ targetTime }: { targetTime: string }) {
  const [remaining, setRemaining] = useState('');
  const [canJoin, setCanJoin] = useState(false);

  useEffect(() => {
    function update() {
      const now = Date.now();
      const target = new Date(targetTime).getTime();
      const diff = target - now;
      const tenMinBefore = diff - 10 * 60 * 1000;

      if (tenMinBefore <= 0) {
        setCanJoin(true);
      }

      if (diff <= 0) {
        setRemaining('Started');
        setCanJoin(true);
        return;
      }

      const hours = Math.floor(diff / (1000 * 60 * 60));
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((diff % (1000 * 60)) / 1000);

      if (hours > 0) {
        setRemaining(`${hours}h ${minutes}m`);
      } else if (minutes > 0) {
        setRemaining(`${minutes}m ${seconds}s`);
      } else {
        setRemaining(`${seconds}s`);
      }
    }
    update();
    const interval = setInterval(update, 1000);
    return () => clearInterval(interval);
  }, [targetTime]);

  return { remaining, canJoin };
}

export function JoinSession({ session, onJoin }: JoinSessionProps) {
  const [cameraOn, setCameraOn] = useState(true);
  const [micOn, setMicOn] = useState(true);
  const { remaining, canJoin } = CountdownTimer({
    targetTime: session.startTime,
  });

  const isLive = session.isActive;

  return (
    <div className="space-y-6">
      {/* Session details */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-2 mb-2">
              {isLive && (
                <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-semibold text-red-700">
                  <span className="h-1.5 w-1.5 rounded-full bg-red-600 animate-pulse" />
                  LIVE
                </span>
              )}
              <span className="text-xs text-[var(--color-text-muted)]">{session.courseTitle}</span>
            </div>
            <h1 className="text-xl font-bold text-[var(--color-text)]">{session.title}</h1>
            {session.description && (
              <p className="mt-1 text-sm text-[var(--color-text-muted)]">{session.description}</p>
            )}
          </div>
          <div className="text-right text-sm text-[var(--color-text-muted)]">
            <p className="font-medium">{session.hostName}</p>
            <p className="text-xs mt-1">
              {new Date(session.startTime).toLocaleDateString('en-US', {
                weekday: 'short',
                month: 'short',
                day: 'numeric',
              })}
            </p>
            <p className="text-xs">
              {new Date(session.startTime).toLocaleTimeString([], {
                hour: '2-digit',
                minute: '2-digit',
              })}{' '}
              -{' '}
              {new Date(session.endTime).toLocaleTimeString([], {
                hour: '2-digit',
                minute: '2-digit',
              })}
            </p>
          </div>
        </div>

        <div className="mt-4 flex items-center gap-4 border-t border-[var(--color-border)] pt-4">
          <div className="flex items-center gap-1.5 text-sm text-[var(--color-text-muted)]">
            <Users className="h-4 w-4" />
            <span>
              {session.participantCount}/{session.maxParticipants}
            </span>
          </div>
          <div className="flex items-center gap-1.5 text-sm text-[var(--color-text-muted)]">
            <Clock className="h-4 w-4" />
            <span>{session.duration} min</span>
          </div>
          {session.isRecording && (
            <span className="rounded-full bg-red-50 px-2.5 py-0.5 text-xs font-medium text-red-600">
              Recording
            </span>
          )}
        </div>
      </div>

      {/* Pre-join check */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <h2 className="text-base font-semibold text-[var(--color-text)] mb-4">Pre-join Check</h2>

        {/* Camera/Mic preview */}
        <div className="aspect-video rounded-lg bg-gray-900 mb-4 flex items-center justify-center">
          {cameraOn ? (
            <div className="text-center text-white">
              <Video className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p className="text-sm opacity-70">Camera preview</p>
            </div>
          ) : (
            <div className="text-center text-white">
              <div className="h-16 w-16 mx-auto mb-2 rounded-full bg-indigo-600 flex items-center justify-center text-2xl font-bold">
                S
              </div>
              <p className="text-sm opacity-70">Camera off</p>
            </div>
          )}
        </div>

        {/* Controls */}
        <div className="flex items-center justify-center gap-3 mb-6">
          <button
            onClick={() => setCameraOn(!cameraOn)}
            className={`flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium transition-colors ${
              cameraOn
                ? 'bg-gray-100 text-[var(--color-text)] hover:bg-gray-200'
                : 'bg-red-100 text-red-700 hover:bg-red-200'
            }`}
            aria-label={cameraOn ? 'Turn camera off' : 'Turn camera on'}
          >
            {cameraOn ? <Video className="h-4 w-4" /> : <VideoOff className="h-4 w-4" />}
            {cameraOn ? 'Camera On' : 'Camera Off'}
          </button>
          <button
            onClick={() => setMicOn(!micOn)}
            className={`flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium transition-colors ${
              micOn
                ? 'bg-gray-100 text-[var(--color-text)] hover:bg-gray-200'
                : 'bg-red-100 text-red-700 hover:bg-red-200'
            }`}
            aria-label={micOn ? 'Turn microphone off' : 'Turn microphone on'}
          >
            {micOn ? <Mic className="h-4 w-4" /> : <MicOff className="h-4 w-4" />}
            {micOn ? 'Mic On' : 'Mic Off'}
          </button>
        </div>

        {/* Join button */}
        {canJoin || isLive ? (
          <button
            onClick={onJoin}
            className="w-full rounded-lg bg-indigo-600 py-3 text-sm font-medium text-white hover:bg-indigo-700 transition-colors flex items-center justify-center gap-2"
          >
            {session.joinUrl ? (
              <>
                Join Session <ExternalLink className="h-4 w-4" />
              </>
            ) : (
              'Join Session'
            )}
          </button>
        ) : (
          <div className="text-center">
            <button
              disabled
              className="w-full rounded-lg bg-gray-200 py-3 text-sm font-medium text-gray-500 cursor-not-allowed"
            >
              Join opens in {remaining}
            </button>
            <p className="mt-2 text-xs text-[var(--color-text-muted)]">
              You can join 10 minutes before the session starts
            </p>
          </div>
        )}
      </div>

      {/* Recording available */}
      {session.recordingUrl && (
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
          <h3 className="text-sm font-semibold text-[var(--color-text)] mb-3">
            Recording Available
          </h3>
          <a
            href={session.recordingUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 text-sm text-indigo-600 hover:text-indigo-700"
          >
            <Video className="h-4 w-4" />
            Watch Recording
          </a>
        </div>
      )}
    </div>
  );
}
