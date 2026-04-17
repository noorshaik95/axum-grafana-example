'use client';

import { useGradeDistribution } from '@/lib/api/hooks';
import { Loader2 } from 'lucide-react';

interface GradeComparisonChartProps {
  courseId: string;
  assignmentId?: string;
}

export function GradeComparisonChart({ courseId, assignmentId }: GradeComparisonChartProps) {
  const { data: distribution, isLoading } = useGradeDistribution(courseId, assignmentId);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--color-text-muted)]" />
      </div>
    );
  }

  if (!distribution) {
    return (
      <div className="py-8 text-center text-sm text-[var(--color-text-muted)]">
        No distribution data available.
      </div>
    );
  }

  const maxCount = Math.max(...distribution.ranges.map((r) => r.count), 1);

  return (
    <div className="space-y-4">
      {/* Percentile badge */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[var(--color-text)]">
            Your Score: {Math.round(distribution.studentScore)}%
          </span>
        </div>
        <span className="rounded-full bg-indigo-100 px-3 py-1 text-xs font-semibold text-indigo-700">
          Top {100 - distribution.studentPercentile}%
        </span>
      </div>

      {/* Bar chart */}
      <div className="space-y-1.5">
        {distribution.ranges.map((range) => {
          const barWidth = (range.count / maxCount) * 100;
          const isStudent = range.isStudent;
          return (
            <div key={range.label} className="flex items-center gap-3">
              <span className="w-14 text-xs text-[var(--color-text-muted)] text-right shrink-0">
                {range.label}
              </span>
              <div className="flex-1 h-6 bg-gray-50 rounded relative">
                <div
                  className={`h-full rounded transition-all ${
                    isStudent ? 'bg-indigo-600' : 'bg-gray-200'
                  }`}
                  style={{ width: `${Math.max(barWidth, 2)}%` }}
                />
                {range.count > 0 && (
                  <span
                    className={`absolute right-2 top-1/2 -translate-y-1/2 text-xs font-medium ${
                      isStudent ? 'text-white' : 'text-[var(--color-text-muted)]'
                    }`}
                  >
                    {range.count}
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Class average */}
      <div className="flex items-center justify-between border-t border-[var(--color-border)] pt-3">
        <span className="text-xs text-[var(--color-text-muted)]">Class Average</span>
        <span className="text-sm font-semibold text-[var(--color-text)]">
          {Math.round(distribution.classAverage)}%
        </span>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-xs text-[var(--color-text-muted)]">Total Students</span>
        <span className="text-sm text-[var(--color-text)]">{distribution.totalStudents}</span>
      </div>
    </div>
  );
}
