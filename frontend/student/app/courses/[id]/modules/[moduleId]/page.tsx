'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Loader2, AlertCircle } from 'lucide-react';
import { useModuleDetail } from '@/lib/api/hooks';
import { ModuleViewer } from '@/components/courses/ModuleViewer';

export default function ModuleViewerPage() {
  const params = useParams();
  const courseId = params.id as string;
  const moduleId = params.moduleId as string;

  const { data: module, isLoading, isError } = useModuleDetail(courseId, moduleId);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--color-text-muted)]" />
      </div>
    );
  }

  if (isError || !module) {
    return (
      <div className="flex flex-col items-center py-20 text-center">
        <AlertCircle className="h-12 w-12 text-[var(--color-error)] mb-3" />
        <h2 className="text-lg font-semibold text-[var(--color-text)]">Module not found</h2>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          This module could not be loaded.
        </p>
        <Link
          href={`/courses/${courseId}`}
          className="mt-4 inline-flex items-center gap-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-700"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Course
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Link
        href={`/courses/${courseId}`}
        className="inline-flex items-center gap-1.5 text-sm font-medium text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Course
      </Link>

      <ModuleViewer courseId={courseId} module={module} />
    </div>
  );
}
