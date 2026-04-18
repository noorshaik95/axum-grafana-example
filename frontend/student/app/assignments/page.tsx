'use client';

import { useState } from 'react';
import Link from 'next/link';
import { ClipboardList, Loader2, AlertCircle } from 'lucide-react';
import { useStudentAssignments } from '@/lib/api/hooks';
import { formatDate } from '@/lib/utils';

type WorkFilter = 'open' | 'done' | 'late';

function getDueBadge(
  dateStr: string,
  status: string
): { label: string; bg: string; color: string } {
  if (status === 'graded' || status === 'submitted') {
    return { label: 'Done', bg: 'var(--forest-50)', color: 'var(--forest-700)' };
  }
  const diffMs = new Date(dateStr).getTime() - Date.now();
  const diffHours = diffMs / (1000 * 60 * 60);
  const diffDays = diffMs / (1000 * 60 * 60 * 24);

  if (diffMs < 0) return { label: 'Overdue', bg: '#fde8e6', color: '#c0392b' };
  if (diffHours < 6)
    return { label: `${Math.round(diffHours)}h left`, bg: '#fff3cd', color: '#7a4900' };
  if (diffDays <= 1) return { label: 'Due tomorrow', bg: '#fff3cd', color: '#7a4900' };
  if (diffDays <= 3)
    return { label: `${Math.ceil(diffDays)}d left`, bg: 'rgba(255,182,72,0.15)', color: '#7a4900' };
  return { label: formatDate(dateStr), bg: 'var(--paper)', color: 'var(--muted)' };
}

const ICONS: Record<string, string> = {
  not_started: '○',
  in_progress: '◑',
  submitted: '●',
  graded: '✓',
};

export default function AllAssignmentsPage() {
  const [filter, setFilter] = useState<WorkFilter>('open');
  const { data, isLoading, isError } = useStudentAssignments();

  const allAssignments = Array.isArray(data) ? data : [];

  const filtered = allAssignments.filter((a) => {
    const isLate =
      new Date(a.dueDate).getTime() < Date.now() &&
      a.status !== 'graded' &&
      a.status !== 'submitted';
    const isDone = a.status === 'graded' || a.status === 'submitted';
    if (filter === 'open') return !isDone && !isLate;
    if (filter === 'done') return isDone;
    if (filter === 'late') return isLate;
    return true;
  });

  const openCount = allAssignments.filter((a) => {
    const isLate =
      new Date(a.dueDate).getTime() < Date.now() &&
      a.status !== 'graded' &&
      a.status !== 'submitted';
    return !isLate && a.status !== 'graded' && a.status !== 'submitted';
  }).length;
  const doneCount = allAssignments.filter(
    (a) => a.status === 'graded' || a.status === 'submitted'
  ).length;
  const lateCount = allAssignments.filter((a) => {
    return (
      new Date(a.dueDate).getTime() < Date.now() &&
      a.status !== 'graded' &&
      a.status !== 'submitted'
    );
  }).length;

  const sorted = [...filtered].sort(
    (a, b) => new Date(a.dueDate).getTime() - new Date(b.dueDate).getTime()
  );

  return (
    <div className="max-w-2xl mx-auto space-y-5">
      <div>
        <h1 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
          Work
        </h1>
        <p className="text-sm mt-1" style={{ color: 'var(--muted)' }}>
          Assignments across all your courses
        </p>
      </div>

      {/* Segment tabs */}
      <div
        className="flex gap-1 p-1 rounded-xl w-fit"
        style={{ background: 'var(--paper)', border: '1px solid var(--border)' }}
      >
        {(
          [
            { key: 'open', label: `Open ${openCount}` },
            { key: 'done', label: `Done ${doneCount}` },
            { key: 'late', label: `Late ${lateCount}` },
          ] as const
        ).map((tab) => {
          const isActive = filter === tab.key;
          return (
            <button
              key={tab.key}
              onClick={() => setFilter(tab.key)}
              className="px-4 py-1.5 rounded-lg text-sm font-medium transition-all"
              style={{
                background: isActive ? '#fff' : 'transparent',
                color: isActive
                  ? tab.key === 'late'
                    ? '#c0392b'
                    : 'var(--forest-700)'
                  : 'var(--muted)',
                boxShadow: isActive ? 'var(--shadow-sm)' : 'none',
                border: isActive ? '1px solid var(--border)' : '1px solid transparent',
              }}
            >
              {tab.label}
            </button>
          );
        })}
      </div>

      <div
        className="rounded-xl overflow-hidden"
        style={{ background: '#fff', border: '1px solid var(--border)' }}
      >
        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2
              style={{ width: 28, height: 28, color: 'var(--muted)' }}
              className="animate-spin"
            />
          </div>
        ) : isError ? (
          <div
            className="flex items-center justify-center gap-2 py-16"
            style={{ color: 'var(--warm)' }}
          >
            <AlertCircle style={{ width: 18, height: 18 }} />
            <span className="text-sm">Failed to load assignments</span>
          </div>
        ) : sorted.length === 0 ? (
          <div className="py-16 text-center">
            <ClipboardList
              style={{ width: 32, height: 32, color: 'var(--muted)', margin: '0 auto 8px' }}
            />
            <p className="text-sm" style={{ color: 'var(--muted)' }}>
              No {filter} assignments.
            </p>
          </div>
        ) : (
          <div className="divide-y" style={{ borderColor: 'var(--border)' }}>
            {sorted.map((assignment) => {
              const badge = getDueBadge(assignment.dueDate, assignment.status);
              const icon = ICONS[assignment.status] ?? '○';
              const isDone = assignment.status === 'graded' || assignment.status === 'submitted';
              return (
                <Link
                  key={assignment.id}
                  href={`/courses/${assignment.courseId}/assignments/${assignment.id}`}
                  className="flex items-center gap-4 p-4 transition-colors hover:bg-[var(--forest-50)]"
                >
                  <span
                    className="shrink-0 mono text-base"
                    style={{
                      color: isDone ? 'var(--forest-500)' : 'var(--muted)',
                      width: 20,
                      textAlign: 'center',
                    }}
                  >
                    {icon}
                  </span>
                  <div className="flex-1 min-w-0">
                    <p
                      className="text-sm font-medium"
                      style={{
                        color: 'var(--ink)',
                        textDecoration: isDone ? 'line-through' : 'none',
                        opacity: isDone ? 0.7 : 1,
                      }}
                    >
                      {assignment.title}
                    </p>
                    <div className="flex items-center gap-2 mt-0.5">
                      <span className="text-xs" style={{ color: 'var(--muted)' }}>
                        {assignment.courseTitle}
                      </span>
                      <span className="text-xs" style={{ color: 'var(--muted)' }}>
                        · {assignment.maxPoints} pts
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    {assignment.status === 'graded' && assignment.grade !== null && (
                      <span
                        className="text-sm font-bold mono"
                        style={{ color: 'var(--forest-600)' }}
                      >
                        {assignment.grade}/{assignment.maxPoints}
                      </span>
                    )}
                    <span
                      className="text-xs font-medium px-2.5 py-1 rounded-full mono"
                      style={{ background: badge.bg, color: badge.color }}
                    >
                      {badge.label}
                    </span>
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
