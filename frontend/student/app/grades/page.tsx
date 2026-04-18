'use client';

import Link from 'next/link';
import {
  GraduationCap,
  BookOpen,
  TrendingUp,
  Loader2,
  AlertCircle,
  ChevronRight,
} from 'lucide-react';
import { useGradesOverview } from '@/lib/api/hooks';

function getLetterGrade(pct: number): string {
  if (pct >= 93) return 'A';
  if (pct >= 90) return 'A-';
  if (pct >= 87) return 'B+';
  if (pct >= 83) return 'B';
  if (pct >= 80) return 'B-';
  if (pct >= 77) return 'C+';
  if (pct >= 73) return 'C';
  if (pct >= 70) return 'C-';
  if (pct >= 60) return 'D';
  return 'F';
}

function getLetterColor(pct: number): string {
  if (pct >= 90) return 'var(--forest-600)';
  if (pct >= 80) return '#2980b9';
  if (pct >= 70) return 'var(--muted)';
  return 'var(--warm)';
}

function getOnTrackNote(pct: number): string {
  if (pct >= 90) return 'On track for A';
  if (pct >= 80) return 'On track for B';
  if (pct >= 70) return 'On track for C';
  return 'Needs attention';
}

function computeGpa(grades: { currentGrade: number }[]): string {
  if (grades.length === 0) return '--';
  const total = grades.reduce((sum, g) => {
    const pct = g.currentGrade;
    const pts =
      pct >= 93
        ? 4.0
        : pct >= 90
          ? 3.7
          : pct >= 87
            ? 3.3
            : pct >= 83
              ? 3.0
              : pct >= 80
                ? 2.7
                : pct >= 77
                  ? 2.3
                  : pct >= 73
                    ? 2.0
                    : pct >= 70
                      ? 1.7
                      : pct >= 60
                        ? 1.0
                        : 0;
    return sum + pts;
  }, 0);
  return (total / grades.length).toFixed(2);
}

export default function GradesPage() {
  const { data: grades, isLoading, isError } = useGradesOverview();

  const gradeList = Array.isArray(grades) ? grades : [];
  const gpa = computeGpa(gradeList);
  const avg =
    gradeList.length > 0
      ? Math.round(gradeList.reduce((s, g) => s + g.currentGrade, 0) / gradeList.length)
      : null;

  // Trend: compare top half avg to bottom half avg
  const sorted = [...gradeList].sort((a, b) => a.currentGrade - b.currentGrade);
  const trendLabel =
    sorted.length >= 2 && sorted[sorted.length - 1].currentGrade > sorted[0].currentGrade
      ? '↑ Improving'
      : sorted.length >= 2
        ? '→ Steady'
        : '--';

  return (
    <div className="max-w-2xl mx-auto space-y-5">
      <div>
        <h1 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
          Grades
        </h1>
        <p className="text-sm mt-1" style={{ color: 'var(--muted)' }}>
          Your academic performance
        </p>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-3 gap-3">
        {[
          {
            label: 'Projection',
            value: avg !== null ? `${avg}%` : '--',
            icon: TrendingUp,
            note: avg !== null ? getLetterGrade(avg) : '',
          },
          {
            label: 'GPA',
            value: isLoading ? null : gpa,
            icon: GraduationCap,
            note: 'Estimate',
          },
          {
            label: 'Trend',
            value: trendLabel,
            icon: BookOpen,
            note: `${gradeList.length} courses`,
          },
        ].map((stat) => (
          <div
            key={stat.label}
            className="rounded-xl p-4"
            style={{ background: '#fff', border: '1px solid var(--border)' }}
          >
            <div className="flex items-center justify-between mb-2">
              <p
                className="text-xs font-semibold uppercase tracking-wide"
                style={{ color: 'var(--muted)' }}
              >
                {stat.label}
              </p>
              <stat.icon style={{ width: 13, height: 13, color: 'var(--muted)' }} />
            </div>
            <p className="text-xl font-bold mono" style={{ color: 'var(--ink)' }}>
              {stat.value === null ? (
                <Loader2 style={{ width: 18, height: 18 }} className="animate-spin" />
              ) : (
                stat.value
              )}
            </p>
            <p className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>
              {stat.note}
            </p>
          </div>
        ))}
      </div>

      {/* Course rows */}
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="h-20 rounded-xl animate-pulse"
              style={{ background: 'var(--paper)' }}
            />
          ))}
        </div>
      ) : isError ? (
        <div
          className="rounded-xl p-12 text-center"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <AlertCircle
            style={{ width: 28, height: 28, color: 'var(--warm)', margin: '0 auto 8px' }}
          />
          <p className="text-sm" style={{ color: 'var(--warm)' }}>
            Failed to load grades
          </p>
        </div>
      ) : gradeList.length === 0 ? (
        <div
          className="rounded-xl p-12 text-center"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <GraduationCap
            style={{ width: 32, height: 32, color: 'var(--muted)', margin: '0 auto 8px' }}
          />
          <p className="text-sm" style={{ color: 'var(--muted)' }}>
            No grades yet. They&apos;ll appear once assignments are graded.
          </p>
        </div>
      ) : (
        <div
          className="rounded-xl overflow-hidden"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <div className="divide-y" style={{ borderColor: 'var(--border)' }}>
            {gradeList.map((g) => {
              const pct = Math.round(g.currentGrade);
              const letterColor = getLetterColor(pct);
              const onTrack = getOnTrackNote(pct);
              return (
                <Link
                  key={g.courseId}
                  href={`/grades/${g.courseId}`}
                  className="flex items-center gap-4 p-4 transition-colors hover:bg-[var(--forest-50)]"
                >
                  <div
                    className="flex items-center justify-center rounded-xl shrink-0 font-bold text-xs"
                    style={{
                      width: 40,
                      height: 40,
                      background: 'var(--forest-50)',
                      border: '1px solid var(--forest-200)',
                      color: 'var(--forest-700)',
                    }}
                  >
                    {g.courseCode?.substring(0, 2) ?? 'CS'}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate" style={{ color: 'var(--ink)' }}>
                      {g.courseTitle}
                    </p>
                    <p className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>
                      {pct}% · {onTrack}
                    </p>
                  </div>
                  <div className="flex items-center gap-3 shrink-0">
                    <span className="text-lg font-bold mono" style={{ color: letterColor }}>
                      {g.letterGrade}
                    </span>
                    <ChevronRight style={{ width: 14, height: 14, color: 'var(--muted)' }} />
                  </div>
                </Link>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
