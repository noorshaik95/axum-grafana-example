'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, FileText, Loader2, AlertCircle } from 'lucide-react';
import { useStudentAssignment } from '@/lib/api/hooks';
import { SubmissionUploader } from '@/components/assignments/SubmissionUploader';
import { formatDate } from '@/lib/utils';

export default function AssignmentSubmissionPage() {
  const params = useParams();
  const courseId = params.id as string;
  const assignmentId = params.assignmentId as string;

  const { data: assignment, isLoading, isError } = useStudentAssignment(assignmentId);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--color-text-muted)]" />
      </div>
    );
  }

  if (isError || !assignment) {
    return (
      <div className="flex flex-col items-center py-20 text-center">
        <AlertCircle className="h-12 w-12 text-[var(--color-error)] mb-3" />
        <h2 className="text-lg font-semibold text-[var(--color-text)]">Assignment not found</h2>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          This assignment could not be loaded.
        </p>
        <Link
          href={`/courses/${courseId}/assignments`}
          className="mt-4 inline-flex items-center gap-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-700"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Assignments
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-3xl">
      <Link
        href={`/courses/${courseId}/assignments`}
        className="inline-flex items-center gap-1.5 text-sm font-medium text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Assignments
      </Link>

      {/* Assignment info */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <div className="flex items-center gap-2 mb-3">
          <FileText className="h-5 w-5 text-indigo-600" />
          <h1 className="text-xl font-bold text-[var(--color-text)]">{assignment.title}</h1>
        </div>
        {assignment.description && (
          <p className="text-sm text-[var(--color-text-muted)] whitespace-pre-wrap">
            {assignment.description}
          </p>
        )}
        <div className="mt-4 grid grid-cols-2 gap-4 border-t border-[var(--color-border)] pt-4 sm:grid-cols-4">
          <div>
            <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase">
              Max Points
            </p>
            <p className="mt-1 text-sm font-semibold text-[var(--color-text)]">
              {assignment.maxPoints}
            </p>
          </div>
          <div>
            <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase">Due Date</p>
            <p className="mt-1 text-sm font-semibold text-[var(--color-text)]">
              {formatDate(assignment.dueDate)}
            </p>
          </div>
          <div>
            <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase">
              Late Penalty
            </p>
            <p className="mt-1 text-sm font-semibold text-[var(--color-text)]">
              {assignment.latePenaltyPercent}%/day
            </p>
          </div>
          <div>
            <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase">
              Max Late Days
            </p>
            <p className="mt-1 text-sm font-semibold text-[var(--color-text)]">
              {assignment.maxLateDays}
            </p>
          </div>
        </div>
      </div>

      {/* Submission area */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <h2 className="text-base font-semibold text-[var(--color-text)] mb-4">Submission</h2>
        <SubmissionUploader assignment={assignment} />
      </div>
    </div>
  );
}
