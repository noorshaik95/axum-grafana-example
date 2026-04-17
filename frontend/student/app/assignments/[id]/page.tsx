'use client';

import { useParams } from 'next/navigation';
import { FileText, Loader2, AlertCircle } from 'lucide-react';
import { useAssignment } from '../../../../shared/lib/api/hooks';

export default function AssignmentDetailPage() {
  const params = useParams();
  const assignmentId = params.id as string;
  const { data: assignment, isLoading, isError } = useAssignment(assignmentId);

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
          The assignment could not be loaded.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-3xl">
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <div className="flex items-center gap-2 mb-3">
          <FileText className="h-5 w-5 text-indigo-600" />
          <h1 className="text-xl font-bold text-[var(--color-text)]">{assignment.title}</h1>
        </div>
        {assignment.description && (
          <p className="text-sm text-[var(--color-text-muted)]">{assignment.description}</p>
        )}
        <div className="mt-4 grid grid-cols-2 gap-4 border-t border-[var(--color-border)] pt-4">
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
              {assignment.dueDate ? new Date(assignment.dueDate).toLocaleDateString() : '--'}
            </p>
          </div>
        </div>
      </div>

      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <h2 className="text-base font-semibold text-[var(--color-text)]">Submission</h2>
        <p className="mt-2 text-sm text-[var(--color-text-muted)]">
          Submit your work for this assignment. File upload and text submissions coming soon.
        </p>
      </div>
    </div>
  );
}
