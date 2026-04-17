'use client';

import { useState, useCallback } from 'react';
import Link from 'next/link';
import {
  CheckCircle2,
  Circle,
  FileText,
  Video,
  ExternalLink,
  ChevronRight,
  ChevronLeft,
  Loader2,
} from 'lucide-react';
import type { CourseModule, ModuleLesson, LessonResource } from '@/lib/api/student';
import { useMarkLessonComplete } from '@/lib/api/hooks';

interface ModuleViewerProps {
  courseId: string;
  module: CourseModule;
}

function ResourceIcon({ contentType }: { contentType: string }) {
  if (contentType.startsWith('video/')) return <Video className="h-4 w-4" />;
  if (contentType === 'application/pdf') return <FileText className="h-4 w-4" />;
  return <ExternalLink className="h-4 w-4" />;
}

function LessonContent({ resource }: { resource: LessonResource }) {
  const contentType = resource.contentType;

  if (contentType.startsWith('video/')) {
    const src = resource.manifestUrl || resource.storageKey;
    return (
      <div className="aspect-video w-full rounded-lg overflow-hidden bg-black">
        <video src={src} controls className="h-full w-full" preload="metadata">
          Your browser does not support the video element.
        </video>
      </div>
    );
  }

  if (contentType === 'application/pdf') {
    const src = resource.manifestUrl || resource.storageKey;
    return (
      <div
        className="w-full rounded-lg overflow-hidden border border-[var(--color-border)]"
        style={{ height: '70vh' }}
      >
        <iframe src={src} className="h-full w-full" title={resource.name} />
      </div>
    );
  }

  // Link / other type
  const url = resource.manifestUrl || resource.storageKey;
  return (
    <div className="rounded-xl border border-[var(--color-border)] bg-white p-6">
      <div className="flex items-center gap-3 mb-4">
        <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600">
          <ExternalLink className="h-6 w-6" />
        </div>
        <div>
          <p className="text-sm font-medium text-[var(--color-text)]">{resource.name}</p>
          {resource.description && (
            <p className="text-xs text-[var(--color-text-muted)]">{resource.description}</p>
          )}
        </div>
      </div>
      <a
        href={url}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors"
      >
        Open Resource <ExternalLink className="h-4 w-4" />
      </a>
    </div>
  );
}

export function ModuleViewer({ courseId, module: mod }: ModuleViewerProps) {
  const lessons = mod.lessons ?? [];
  const [activeLessonIdx, setActiveLessonIdx] = useState(0);
  const markComplete = useMarkLessonComplete();

  const activeLesson = lessons[activeLessonIdx] as ModuleLesson | undefined;
  const activeResource = activeLesson?.resources?.[0] as LessonResource | undefined;

  const handleMarkComplete = useCallback(() => {
    if (!activeLesson) return;
    markComplete.mutate({ courseId, lessonId: activeLesson.id });
  }, [courseId, activeLesson, markComplete]);

  const goNext = () => {
    if (activeLessonIdx < lessons.length - 1) {
      setActiveLessonIdx(activeLessonIdx + 1);
    }
  };

  const goPrev = () => {
    if (activeLessonIdx > 0) {
      setActiveLessonIdx(activeLessonIdx - 1);
    }
  };

  return (
    <div className="flex flex-col gap-6 lg:flex-row">
      {/* Lesson Tree - Left sidebar */}
      <aside className="w-full lg:w-72 shrink-0">
        <div className="rounded-xl border border-[var(--color-border)] bg-white shadow-sm">
          <div className="p-4 border-b border-[var(--color-border)]">
            <h3 className="text-sm font-semibold text-[var(--color-text)]">{mod.name}</h3>
            <p className="text-xs text-[var(--color-text-muted)] mt-1">
              {lessons.filter((l) => l.completed).length}/{lessons.length} completed
            </p>
          </div>
          <nav className="p-2 space-y-0.5 max-h-[60vh] overflow-y-auto" aria-label="Lessons">
            {lessons.map((lesson, idx) => (
              <button
                key={lesson.id}
                onClick={() => setActiveLessonIdx(idx)}
                className={`flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-sm transition-colors ${
                  idx === activeLessonIdx
                    ? 'bg-indigo-50 text-indigo-700 font-medium'
                    : 'text-[var(--color-text)] hover:bg-[var(--color-bg-muted)]'
                }`}
                data-testid={
                  lesson.completed
                    ? `lesson-completed-${lesson.name.replace(/\s+/g, '-')}`
                    : undefined
                }
              >
                {lesson.completed ? (
                  <CheckCircle2 className="h-4 w-4 shrink-0 text-green-600" />
                ) : (
                  <Circle className="h-4 w-4 shrink-0 text-[var(--color-text-muted)]" />
                )}
                <span className="truncate">{lesson.name}</span>
              </button>
            ))}
          </nav>
        </div>
      </aside>

      {/* Content area */}
      <div className="flex-1 min-w-0 space-y-4">
        {activeLesson ? (
          <>
            <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
              <h2 className="text-lg font-semibold text-[var(--color-text)]">
                {activeLesson.name}
              </h2>
              {activeLesson.description && (
                <p className="mt-1 text-sm text-[var(--color-text-muted)]">
                  {activeLesson.description}
                </p>
              )}
            </div>

            {activeResource ? (
              <LessonContent resource={activeResource} />
            ) : (
              <div className="rounded-xl border border-[var(--color-border)] bg-white p-8 text-center">
                <FileText className="mx-auto h-8 w-8 text-[var(--color-text-muted)]" />
                <p className="mt-2 text-sm text-[var(--color-text-muted)]">
                  No content available for this lesson.
                </p>
              </div>
            )}

            {/* Actions */}
            <div className="flex items-center justify-between rounded-xl border border-[var(--color-border)] bg-white p-4 shadow-sm">
              <button
                onClick={goPrev}
                disabled={activeLessonIdx === 0}
                className="inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)] disabled:opacity-40 transition-colors"
              >
                <ChevronLeft className="h-4 w-4" />
                Previous
              </button>

              {!activeLesson.completed ? (
                <button
                  onClick={handleMarkComplete}
                  disabled={markComplete.isPending}
                  className="inline-flex items-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50 transition-colors"
                >
                  {markComplete.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <CheckCircle2 className="h-4 w-4" />
                  )}
                  Mark as Complete
                </button>
              ) : (
                <span className="inline-flex items-center gap-1.5 text-sm text-green-600 font-medium">
                  <CheckCircle2 className="h-4 w-4" />
                  Completed
                </span>
              )}

              <button
                onClick={goNext}
                disabled={activeLessonIdx === lessons.length - 1}
                className="inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)] disabled:opacity-40 transition-colors"
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </>
        ) : (
          <div className="rounded-xl border border-[var(--color-border)] bg-white p-8 text-center">
            <p className="text-sm text-[var(--color-text-muted)]">
              Select a lesson to view its content.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
