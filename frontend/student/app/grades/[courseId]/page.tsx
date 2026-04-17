'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Loader2, AlertCircle, GraduationCap } from 'lucide-react';
import { useCourseGrades, useGradesOverview } from '@/lib/api/hooks';
import { GradeComparisonChart } from '@/components/grades/GradeComparisonChart';
import { formatDate, getGradeColor } from '@/lib/utils';

const statusBadge: Record<string, string> = {
  published: 'bg-green-50 text-green-700',
  draft: 'bg-amber-50 text-amber-700',
  ungraded: 'bg-gray-100 text-gray-600',
};

export default function CourseGradeBreakdownPage() {
  const params = useParams();
  const courseId = params.courseId as string;

  const { data: courseGrades, isLoading, isError } = useCourseGrades(courseId);
  const { data: overview } = useGradesOverview();

  const gradeList = Array.isArray(courseGrades) ? courseGrades : [];
  const overviewList = Array.isArray(overview) ? overview : [];
  const courseSummary = overviewList.find((g) => g.courseId === courseId);

  return (
    <div className="space-y-6">
      <Link
        href="/grades"
        className="inline-flex items-center gap-1.5 text-sm font-medium text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Grades
      </Link>

      {/* Course summary */}
      {courseSummary && (
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
          <div className="flex items-start justify-between">
            <div>
              <div className="flex items-center gap-2 mb-2">
                <span className="rounded-md bg-indigo-600 px-3 py-1 text-sm font-semibold text-white">
                  {courseSummary.courseCode}
                </span>
              </div>
              <h1 className="text-xl font-bold text-[var(--color-text)]">
                {courseSummary.courseTitle}
              </h1>
            </div>
            <div className="text-right">
              <p className={`text-3xl font-bold ${getGradeColor(courseSummary.currentGrade)}`}>
                {courseSummary.letterGrade}
              </p>
              <p className="text-sm text-[var(--color-text-muted)]">
                {Math.round(courseSummary.currentGrade)}%
              </p>
            </div>
          </div>

          {/* Category breakdown */}
          {courseSummary.breakdown && courseSummary.breakdown.length > 0 && (
            <div className="mt-5 border-t border-[var(--color-border)] pt-4">
              <h3 className="text-sm font-medium text-[var(--color-text)] mb-3">Grade Breakdown</h3>
              <div className="space-y-2">
                {courseSummary.breakdown.map((cat) => (
                  <div key={cat.category} className="flex items-center gap-3">
                    <span className="w-28 text-xs text-[var(--color-text-muted)] capitalize truncate">
                      {cat.category} ({cat.weight}%)
                    </span>
                    <div className="flex-1 h-2 rounded-full bg-gray-100">
                      <div
                        className="h-2 rounded-full bg-indigo-600 transition-all"
                        style={{ width: `${Math.min(cat.percentage, 100)}%` }}
                      />
                    </div>
                    <span className="w-12 text-xs font-medium text-[var(--color-text)] text-right">
                      {Math.round(cat.percentage)}%
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Assignment grades table */}
        <div className="lg:col-span-2 rounded-xl border border-[var(--color-border)] bg-white shadow-sm">
          <div className="p-5 border-b border-[var(--color-border)]">
            <h2 className="text-base font-semibold text-[var(--color-text)]">Assignment Grades</h2>
          </div>
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-8 w-8 animate-spin text-[var(--color-text-muted)]" />
            </div>
          ) : isError ? (
            <div className="flex items-center justify-center gap-2 py-16 text-[var(--color-error)]">
              <AlertCircle className="h-5 w-5" />
              <span className="text-sm">Failed to load grades</span>
            </div>
          ) : gradeList.length === 0 ? (
            <div className="py-16 text-center">
              <GraduationCap className="mx-auto h-8 w-8 text-[var(--color-text-muted)]" />
              <p className="mt-2 text-sm text-[var(--color-text-muted)]">
                No graded assignments yet.
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-[var(--color-border)] bg-[var(--color-bg-muted)]">
                    <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                      Assignment
                    </th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                      Category
                    </th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                      Score
                    </th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                      Date
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {gradeList.map((grade) => {
                    const pct =
                      grade.score !== null && grade.maxPoints > 0
                        ? Math.round((grade.score / grade.maxPoints) * 100)
                        : null;
                    return (
                      <tr
                        key={grade.assignmentId}
                        className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-muted)]/50 transition-colors"
                      >
                        <td className="px-5 py-3 text-sm text-[var(--color-text)]">
                          {grade.assignmentTitle}
                        </td>
                        <td className="px-5 py-3 text-xs text-[var(--color-text-muted)] capitalize">
                          {grade.category}
                        </td>
                        <td className="px-5 py-3">
                          {grade.score !== null ? (
                            <span className={`text-sm font-medium ${getGradeColor(pct ?? 0)}`}>
                              {grade.score}/{grade.maxPoints}
                              {grade.adjustedScore !== null &&
                                grade.adjustedScore !== grade.score && (
                                  <span className="text-xs text-[var(--color-text-muted)] ml-1">
                                    (adj: {grade.adjustedScore})
                                  </span>
                                )}
                            </span>
                          ) : (
                            <span className="text-sm text-[var(--color-text-muted)]">--</span>
                          )}
                        </td>
                        <td className="px-5 py-3">
                          <span
                            className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                              statusBadge[grade.status] ?? statusBadge.ungraded
                            }`}
                          >
                            {grade.status}
                          </span>
                        </td>
                        <td className="px-5 py-3 text-sm text-[var(--color-text-muted)]">
                          {grade.gradedAt ? formatDate(grade.gradedAt) : '--'}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Class comparison chart */}
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
          <h3 className="text-sm font-semibold text-[var(--color-text)] mb-4">Class Comparison</h3>
          <GradeComparisonChart courseId={courseId} />
        </div>
      </div>
    </div>
  );
}
