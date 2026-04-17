'use client';

import { useState } from 'react';
import Link from 'next/link';
import { ClipboardList, Loader2, AlertCircle, Clock, Filter } from 'lucide-react';
import { useStudentAssignments } from '@/lib/api/hooks';
import { formatDate } from '@/lib/utils';

type StatusFilter = 'all' | 'not_started' | 'in_progress' | 'submitted' | 'graded';

const statusColors: Record<string, string> = {
  not_started: 'bg-gray-100 text-gray-600',
  in_progress: 'bg-amber-50 text-amber-700',
  submitted: 'bg-blue-50 text-blue-700',
  graded: 'bg-green-50 text-green-700',
};

function dueUrgencyClass(dateStr: string): string {
  const diffMs = new Date(dateStr).getTime() - Date.now();
  const diffDays = diffMs / (1000 * 60 * 60 * 24);
  if (diffMs < 0) return 'text-red-600 font-semibold';
  if (diffDays <= 1) return 'text-red-500 font-medium';
  if (diffDays <= 3) return 'text-amber-500 font-medium';
  return 'text-[var(--color-text-muted)]';
}

export default function AllAssignmentsPage() {
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const { data, isLoading, isError } = useStudentAssignments();

  const allAssignments = Array.isArray(data) ? data : [];
  const assignments =
    statusFilter === 'all'
      ? allAssignments
      : allAssignments.filter((a) => a.status === statusFilter);

  // Sort by due date ascending
  const sorted = [...assignments].sort(
    (a, b) => new Date(a.dueDate).getTime() - new Date(b.dueDate).getTime()
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">All Assignments</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Assignments across all your courses, sorted by deadline
        </p>
      </div>

      {/* Filter */}
      <div className="flex items-center gap-2 flex-wrap">
        <Filter className="h-4 w-4 text-[var(--color-text-muted)]" />
        {(['all', 'not_started', 'in_progress', 'submitted', 'graded'] as const).map((status) => (
          <button
            key={status}
            onClick={() => setStatusFilter(status)}
            className={`rounded-lg px-3 py-1.5 text-xs font-medium capitalize transition-colors ${
              statusFilter === status
                ? 'bg-indigo-600 text-white'
                : 'bg-white border border-[var(--color-border)] text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)]'
            }`}
          >
            {status.replace('_', ' ')}
          </button>
        ))}
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
        ) : sorted.length === 0 ? (
          <div className="py-16 text-center">
            <ClipboardList className="mx-auto h-10 w-10 text-[var(--color-text-muted)]" />
            <p className="mt-3 text-sm text-[var(--color-text-muted)]">
              {statusFilter === 'all'
                ? 'No assignments yet.'
                : `No ${statusFilter.replace('_', ' ')} assignments.`}
            </p>
          </div>
        ) : (
          <div className="divide-y divide-[var(--color-border)]">
            {sorted.map((assignment) => (
              <Link
                key={assignment.id}
                href={`/courses/${assignment.courseId}/assignments/${assignment.id}`}
                className="flex items-center justify-between p-5 hover:bg-[var(--color-bg-muted)]/50 transition-colors"
              >
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-[var(--color-text)]">{assignment.title}</p>
                  <div className="mt-1 flex items-center gap-3 text-xs text-[var(--color-text-muted)]">
                    <span>{assignment.courseTitle}</span>
                    <span>{assignment.maxPoints} pts</span>
                  </div>
                </div>
                <div className="ml-3 flex items-center gap-3">
                  <div
                    className={`flex items-center gap-1 text-xs ${dueUrgencyClass(assignment.dueDate)}`}
                  >
                    <Clock className="h-3 w-3" />
                    {formatDate(assignment.dueDate)}
                  </div>
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
