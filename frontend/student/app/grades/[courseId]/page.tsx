'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Loader2, AlertCircle, GraduationCap } from 'lucide-react';
import { useCourseGrades, useGradesOverview } from '@/lib/api/hooks';
import { GradeComparisonChart } from '@/components/grades/GradeComparisonChart';
import { formatDate } from '@/lib/utils';

const statusBadge: Record<string, { bg: string; color: string }> = {
  published: { bg: 'var(--forest-50)', color: 'var(--forest-700)' },
  draft: { bg: 'rgba(255,182,72,0.15)', color: '#7a4900' },
  ungraded: { bg: 'var(--paper)', color: 'var(--muted)' },
};

function getLetterColor(pct: number): string {
  if (pct >= 90) return 'var(--forest-600)';
  if (pct >= 80) return '#2980b9';
  if (pct >= 70) return 'var(--muted)';
  return 'var(--warm)';
}

function getPathToAGuidance(pct: number): { text: string } {
  if (pct >= 93) return { text: "You're on track for an A! Keep it up." };
  if (pct >= 85) return { text: 'A few strong assignments will push you to an A.' };
  if (pct >= 75) return { text: 'Focus on upcoming assignments to reach an A.' };
  return { text: "There's still time to turn this around. Aim for 90%+ on remaining work." };
}

export default function CourseGradeBreakdownPage() {
  const params = useParams();
  const courseId = params.courseId as string;

  const { data: courseGrades, isLoading, isError } = useCourseGrades(courseId);
  const { data: overview } = useGradesOverview();

  const gradeList = Array.isArray(courseGrades) ? courseGrades : [];
  const overviewList = Array.isArray(overview) ? overview : [];
  const courseSummary = overviewList.find((g) => g.courseId === courseId);
  const currentPct = courseSummary ? Math.round(courseSummary.currentGrade) : 0;
  const pathToA = getPathToAGuidance(currentPct);
  const pathPct = Math.min((currentPct / 93) * 100, 100);

  return (
    <div className="max-w-3xl mx-auto space-y-5">
      <Link
        href="/grades"
        className="inline-flex items-center gap-1.5 text-sm"
        style={{ color: 'var(--muted)' }}
      >
        <ArrowLeft style={{ width: 14, height: 14 }} />
        Grades
      </Link>

      {/* Course summary */}
      {courseSummary && (
        <div
          className="rounded-xl p-6"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <div className="flex items-start justify-between gap-4">
            <div>
              <span
                className="text-xs font-bold px-2.5 py-1 rounded"
                style={{ background: 'var(--forest-700)', color: '#fff' }}
              >
                {courseSummary.courseCode}
              </span>
              <h1 className="serif text-xl sm:text-2xl mt-3" style={{ color: 'var(--ink)' }}>
                {courseSummary.courseTitle}
              </h1>
            </div>
            <div className="text-right shrink-0">
              <p className="text-3xl font-bold mono" style={{ color: getLetterColor(currentPct) }}>
                {courseSummary.letterGrade}
              </p>
              <p className="text-sm" style={{ color: 'var(--muted)' }}>
                {currentPct}%
              </p>
            </div>
          </div>

          {/* Path to A */}
          <div
            className="mt-5 rounded-xl p-4"
            style={{ background: 'var(--forest-50)', border: '1px solid var(--forest-200)' }}
          >
            <div className="flex items-center justify-between mb-2">
              <p className="text-xs font-semibold" style={{ color: 'var(--forest-700)' }}>
                Path to A
              </p>
              <span className="text-xs mono font-bold" style={{ color: 'var(--forest-600)' }}>
                {currentPct}% / 93%
              </span>
            </div>
            <div
              className="h-2 rounded-full overflow-hidden mb-2"
              style={{ background: 'var(--forest-200)' }}
            >
              <div
                className="h-2 rounded-full transition-all"
                style={{ width: `${pathPct}%`, background: 'var(--forest-600)' }}
              />
            </div>
            <p className="text-xs" style={{ color: 'var(--forest-700)' }}>
              {pathToA.text}
            </p>
          </div>

          {/* Category breakdown */}
          {courseSummary.breakdown && courseSummary.breakdown.length > 0 && (
            <div className="mt-5 pt-4 border-t" style={{ borderColor: 'var(--border)' }}>
              <h3
                className="text-xs font-semibold uppercase tracking-wide mb-3"
                style={{ color: 'var(--muted)' }}
              >
                Grade breakdown
              </h3>
              <div className="space-y-2">
                {courseSummary.breakdown.map((cat) => (
                  <div key={cat.category} className="flex items-center gap-3">
                    <span
                      className="w-28 text-xs capitalize truncate"
                      style={{ color: 'var(--muted)' }}
                    >
                      {cat.category} ({cat.weight}%)
                    </span>
                    <div
                      className="flex-1 h-2 rounded-full overflow-hidden"
                      style={{ background: 'var(--forest-100)' }}
                    >
                      <div
                        className="h-2 rounded-full"
                        style={{
                          width: `${Math.min(cat.percentage, 100)}%`,
                          background: 'var(--forest-600)',
                        }}
                      />
                    </div>
                    <span
                      className="w-10 text-xs font-medium mono text-right"
                      style={{ color: 'var(--ink)' }}
                    >
                      {Math.round(cat.percentage)}%
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      <div className="grid gap-5 lg:grid-cols-3">
        {/* Assignment rows */}
        <div
          className="lg:col-span-2 rounded-xl overflow-hidden"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <div className="p-5 border-b" style={{ borderColor: 'var(--border)' }}>
            <h2 className="text-sm font-semibold" style={{ color: 'var(--ink)' }}>
              Assignment Grades
            </h2>
          </div>
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2
                style={{ width: 24, height: 24, color: 'var(--muted)' }}
                className="animate-spin"
              />
            </div>
          ) : isError ? (
            <div
              className="flex items-center justify-center gap-2 py-16"
              style={{ color: 'var(--warm)' }}
            >
              <AlertCircle style={{ width: 18, height: 18 }} />
              <span className="text-sm">Failed to load grades</span>
            </div>
          ) : gradeList.length === 0 ? (
            <div className="py-16 text-center">
              <GraduationCap
                style={{ width: 28, height: 28, color: 'var(--muted)', margin: '0 auto 8px' }}
              />
              <p className="text-sm" style={{ color: 'var(--muted)' }}>
                No graded assignments yet.
              </p>
            </div>
          ) : (
            <div className="divide-y" style={{ borderColor: 'var(--border)' }}>
              {gradeList.map((grade) => {
                const pct =
                  grade.score !== null && grade.maxPoints > 0
                    ? Math.round((grade.score / grade.maxPoints) * 100)
                    : null;
                const badge = statusBadge[grade.status] ?? statusBadge.ungraded;
                return (
                  <div
                    key={grade.assignmentId}
                    className="flex items-center gap-4 p-4 hover:bg-[var(--forest-50)] transition-colors"
                  >
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate" style={{ color: 'var(--ink)' }}>
                        {grade.assignmentTitle}
                      </p>
                      <p className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>
                        {grade.category}
                        {grade.gradedAt ? ` · ${formatDate(grade.gradedAt)}` : ''}
                      </p>
                    </div>
                    <div className="flex items-center gap-3 shrink-0">
                      {grade.score !== null ? (
                        <span
                          className="text-sm font-bold mono"
                          style={{ color: pct !== null ? getLetterColor(pct) : 'var(--muted)' }}
                        >
                          {grade.score}/{grade.maxPoints}
                        </span>
                      ) : (
                        <span className="text-sm mono" style={{ color: 'var(--muted)' }}>
                          --
                        </span>
                      )}
                      <span
                        className="text-xs px-2 py-0.5 rounded-full font-medium"
                        style={{ background: badge.bg, color: badge.color }}
                      >
                        {grade.status}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Class comparison */}
        <div
          className="rounded-xl p-5"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <h3
            className="text-xs font-semibold uppercase tracking-wide mb-4"
            style={{ color: 'var(--muted)' }}
          >
            Class comparison
          </h3>
          <GradeComparisonChart courseId={courseId} />
        </div>
      </div>
    </div>
  );
}
