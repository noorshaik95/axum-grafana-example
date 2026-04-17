'use client';

import Link from 'next/link';
import {
  GraduationCap,
  BookOpen,
  BarChart3,
  Loader2,
  AlertCircle,
  ChevronRight,
} from 'lucide-react';
import { useGradesOverview } from '@/lib/api/hooks';
import { getGradeLetter, getGradeColor } from '@/lib/utils';

export default function GradesPage() {
  const { data: grades, isLoading, isError } = useGradesOverview();

  const gradeList = Array.isArray(grades) ? grades : [];

  // Calculate cumulative GPA estimate
  const gpa =
    gradeList.length > 0
      ? (
          gradeList.reduce((sum, g) => {
            const gradePoint =
              g.currentGrade >= 90
                ? 4.0
                : g.currentGrade >= 80
                  ? 3.0
                  : g.currentGrade >= 70
                    ? 2.0
                    : g.currentGrade >= 60
                      ? 1.0
                      : 0.0;
            return sum + gradePoint;
          }, 0) / gradeList.length
        ).toFixed(2)
      : '--';

  const totalGraded = gradeList.reduce((sum, g) => sum + g.completedAssignments, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">Grades</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Track your academic performance across all courses
        </p>
      </div>

      {/* Summary stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-[var(--color-text-muted)]">GPA Estimate</p>
            <GraduationCap className="h-4 w-4 text-[var(--color-text-muted)]" />
          </div>
          <p className="mt-2 text-2xl font-bold text-[var(--color-text)]">
            {isLoading ? <Loader2 className="h-6 w-6 animate-spin" /> : gpa}
          </p>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">Cumulative</p>
        </div>
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-[var(--color-text-muted)]">Courses</p>
            <BookOpen className="h-4 w-4 text-[var(--color-text-muted)]" />
          </div>
          <p className="mt-2 text-2xl font-bold text-[var(--color-text)]">
            {isLoading ? <Loader2 className="h-6 w-6 animate-spin" /> : gradeList.length}
          </p>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">Active</p>
        </div>
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-[var(--color-text-muted)]">Graded Items</p>
            <BarChart3 className="h-4 w-4 text-[var(--color-text-muted)]" />
          </div>
          <p className="mt-2 text-2xl font-bold text-[var(--color-text)]">
            {isLoading ? <Loader2 className="h-6 w-6 animate-spin" /> : totalGraded}
          </p>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">Total</p>
        </div>
      </div>

      {/* Course grade cards */}
      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className="h-40 rounded-xl border border-[var(--color-border)] bg-white animate-pulse"
            />
          ))}
        </div>
      ) : isError ? (
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-12 text-center shadow-sm">
          <AlertCircle className="mx-auto h-8 w-8 text-[var(--color-error)]" />
          <p className="mt-2 text-sm text-[var(--color-error)]">Failed to load grades</p>
        </div>
      ) : gradeList.length === 0 ? (
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-12 text-center shadow-sm">
          <GraduationCap className="mx-auto h-10 w-10 text-[var(--color-text-muted)]" />
          <p className="mt-3 text-sm text-[var(--color-text-muted)]">
            No grades yet. Your grades will appear here once assignments are graded.
          </p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {gradeList.map((g) => (
            <Link
              key={g.courseId}
              href={`/grades/${g.courseId}`}
              className="group rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm hover:shadow-md transition-shadow"
            >
              <div className="flex items-start justify-between">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="rounded-md bg-indigo-600 px-2 py-0.5 text-xs font-semibold text-white">
                      {g.courseCode}
                    </span>
                  </div>
                  <p className="mt-2 text-sm font-medium text-[var(--color-text)] truncate group-hover:text-indigo-600 transition-colors">
                    {g.courseTitle}
                  </p>
                </div>
                <div className="text-right shrink-0 ml-3">
                  <p className={`text-2xl font-bold ${getGradeColor(g.currentGrade)}`}>
                    {g.letterGrade}
                  </p>
                  <p className="text-xs text-[var(--color-text-muted)]">
                    {Math.round(g.currentGrade)}%
                  </p>
                </div>
              </div>

              {/* Progress bar */}
              <div className="mt-3">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs text-[var(--color-text-muted)]">
                    {g.completedAssignments}/{g.totalAssignments} graded
                  </span>
                  <span className="text-xs text-[var(--color-text-muted)]">
                    {g.earnedPoints}/{g.totalPoints} pts
                  </span>
                </div>
                <div className="h-1.5 w-full rounded-full bg-gray-100">
                  <div
                    className="h-1.5 rounded-full bg-indigo-600 transition-all"
                    style={{
                      width: `${
                        g.totalAssignments > 0
                          ? (g.completedAssignments / g.totalAssignments) * 100
                          : 0
                      }%`,
                    }}
                  />
                </div>
              </div>

              {/* Category breakdown preview */}
              {g.breakdown && g.breakdown.length > 0 && (
                <div className="mt-3 flex flex-wrap gap-2">
                  {g.breakdown.slice(0, 3).map((cat) => (
                    <span
                      key={cat.category}
                      className="rounded-full bg-gray-50 px-2 py-0.5 text-xs text-[var(--color-text-muted)]"
                    >
                      {cat.category} {cat.weight}%
                    </span>
                  ))}
                </div>
              )}

              <div className="mt-3 flex items-center justify-end text-xs font-medium text-indigo-600 opacity-0 group-hover:opacity-100 transition-opacity">
                View Details <ChevronRight className="h-3 w-3 ml-0.5" />
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
