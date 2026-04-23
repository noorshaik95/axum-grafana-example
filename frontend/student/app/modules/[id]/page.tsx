'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import { useRef } from 'react';
import { AlertCircle, ArrowLeft, ExternalLink, FileText, Loader2 } from 'lucide-react';
import { useContent, useUpdatePosition } from '@/lib/api/hooks';
import { VideoPlayer } from '../../../../shared/components/video-player';

export default function ModuleViewerPage() {
  const params = useParams();
  const contentId = params.id as string;
  const { data: content, isLoading, isError } = useContent(contentId);
  const updatePosition = useUpdatePosition();
  const lastSentRef = useRef<number>(-1);

  const handlePositionChange = (sec: number) => {
    const rounded = Math.round(sec);
    if (rounded === lastSentRef.current) return;
    lastSentRef.current = rounded;
    updatePosition.mutate({ contentId, positionSeconds: rounded });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-7 w-7 animate-spin" style={{ color: 'var(--muted)' }} />
      </div>
    );
  }

  if (isError || !content) {
    return (
      <div className="flex flex-col items-center py-20 text-center">
        <AlertCircle className="h-10 w-10 mb-3" style={{ color: 'var(--warm)' }} />
        <h2 className="text-lg font-semibold" style={{ color: 'var(--ink)' }}>
          Content unavailable
        </h2>
        <p className="mt-1 text-sm" style={{ color: 'var(--muted)' }}>
          This lesson could not be loaded.
        </p>
        <Link
          href="/courses"
          className="mt-4 inline-flex items-center gap-1.5 text-sm font-medium"
          style={{ color: 'var(--forest-700)' }}
        >
          <ArrowLeft className="h-4 w-4" />
          Back to courses
        </Link>
      </div>
    );
  }

  const backHref = content.courseId ? `/courses/${content.courseId}` : '/courses';
  const isVideo = content.contentType.startsWith('video/');
  const isPdf = content.contentType === 'application/pdf';
  const src = content.manifestUrl || content.storageKey;

  return (
    <div className="max-w-3xl mx-auto space-y-4">
      <Link
        href={backHref}
        className="inline-flex items-center gap-1.5 text-sm"
        style={{ color: 'var(--muted)' }}
      >
        <ArrowLeft style={{ width: 14, height: 14 }} />
        Back to course
      </Link>

      <div
        className="rounded-xl p-5"
        style={{ background: '#fff', border: '1px solid var(--border)' }}
      >
        <h1 className="serif text-xl sm:text-2xl leading-snug" style={{ color: 'var(--ink)' }}>
          {content.name}
        </h1>
        {content.description ? (
          <p className="mt-1 text-sm" style={{ color: 'var(--muted)' }}>
            {content.description}
          </p>
        ) : null}
      </div>

      {isVideo ? (
        <VideoPlayer
          src={src}
          posterUrl={content.posterUrl ?? undefined}
          resumePositionSeconds={content.resumePositionSeconds}
          onPositionChange={handlePositionChange}
          tracks={content.tracks?.map((t: NonNullable<typeof content.tracks>[number]) => ({
            src: t.src,
            kind: t.kind as 'subtitles' | 'captions' | undefined,
            srcLang: t.srcLang,
            label: t.label,
            default: t.default,
          }))}
        />
      ) : isPdf ? (
        <div
          className="w-full rounded-xl overflow-hidden"
          style={{ height: '70vh', border: '1px solid var(--border)' }}
        >
          <iframe src={src} className="h-full w-full" title={content.name} />
        </div>
      ) : (
        <div
          className="rounded-xl p-6"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <div className="flex items-center gap-3 mb-4">
            <div
              className="flex h-12 w-12 items-center justify-center rounded-lg"
              style={{ background: 'var(--forest-50)', color: 'var(--forest-700)' }}
            >
              <FileText className="h-6 w-6" />
            </div>
            <div>
              <p className="text-sm font-medium" style={{ color: 'var(--ink)' }}>
                {content.name}
              </p>
              <p className="text-xs" style={{ color: 'var(--muted)' }}>
                {content.contentType}
              </p>
            </div>
          </div>
          <a
            href={src}
            target="_blank"
            rel="noopener noreferrer"
            className="btn-primary inline-flex items-center gap-2"
            style={{ fontSize: 13 }}
          >
            Open resource <ExternalLink className="h-4 w-4" />
          </a>
        </div>
      )}
    </div>
  );
}
