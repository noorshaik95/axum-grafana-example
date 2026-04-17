'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, ClipboardList, Clock, CheckCircle2, Loader2, AlertCircle } from 'lucide-react';
import { useCourse } from '../../../../../shared/lib/api/hooks';
import { useStudentAssignments } from '@/lib/api/hooks';
import { formatDate } from '@/lib/utils';

const statusColors: Record<string, string> = {
  not_started: 'bg-gray-100 text-gray-600',
  in_progress: 'bg-amber-50 text-amber-700',
  submitted: 'bg-blue-50 text-blue-700',
  graded: 'bg-green-50 text-green-700',
};

export default function CourseAssignmentsPage() {
  const params = useParams();
  const courseId = params.id as string;

  const { data: course } = useCourse(courseId);
  const {
    data: assignments,
    isLoading,
    isError,
  } = useStudentAssignments({
    courseId,
  });

  const assignmentList = Array.isArray(assignments) ? assignments : [];

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          href={`/courses/${courseId}`}
          className="inline-flex items-center gap-1.5 text-sm font-medium text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-[var(--color-text)]">Assignments</h1>
          <p className="text-sm text-[var(--color-text-muted)]">{course?.title ?? 'Course'}</p>
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
            <span className="text-sm">Failed to load assignments</span>
          </div>
        ) : assignmentList.length === 0 ? (
          <div className="py-16 text-center">
            <ClipboardList className="mx-auto h-10 w-10 text-[var(--color-text-muted)]" />
            <p className="mt-3 text-sm text-[var(--color-text-muted)]">
              No assignments for this course yet.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-[var(--color-border)]">
            {assignmentList.map((assignment) => (
              <Link
                key={assignment.id}
                href={`/courses/${courseId}/assignments/${assignment.id}`}
                className="flex items-center justify-between p-5 hover:bg-[var(--color-bg-muted)]/50 transition-colors"
              >
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-[var(--color-text)]">{assignment.title}</p>
                  <div className="mt-1 flex items-center gap-3 text-xs text-[var(--color-text-muted)]">
                    <span className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      Due {formatDate(assignment.dueDate)}
                    </span>
                    <span>{assignment.maxPoints} pts</span>
                  </div>
                </div>
                <div className="ml-3 flex items-center gap-3">
                  {assignment.status === 'graded' && assignment.grade !== null && (
                    <span className="text-sm font-semibold text-[var(--color-text)]">
                      {assignment.grade}/{assignment.maxPoints}
                    </span>
                  )}
                  <span
                    className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${
                      statusColors[assignment.status] ?? statusColors.not_started
                    }`}
                  >
                    {assignment.status.replace('_', ' ')}
                  </span>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
