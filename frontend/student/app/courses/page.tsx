'use client';

import { useState } from 'react';
import Link from 'next/link';
import { BookOpen, Loader2, ChevronRight } from 'lucide-react';
import {
  useCourses,
  useMyEnrollments,
  useProfile,
  useEnrollInCourse,
} from '../../../shared/lib/api/hooks';
import { useGradesOverview } from '@/lib/api/hooks';

type TabFilter = 'all' | 'active' | 'archive';

export default function CoursesPage() {
  const [tabFilter, setTabFilter] = useState<TabFilter>('all');

  const { data: profile } = useProfile();
  const studentId = profile?.id ?? '';
  const { data: coursesData, isLoading, isError } = useCourses();
  const { data: enrollments } = useMyEnrollments(studentId);
  const { data: gradesData } = useGradesOverview();
  const enrollMutation = useEnrollInCourse();

  const courses = coursesData?.data ?? [];
  const enrollmentList = Array.isArray(enrollments) ? enrollments : [];
  const gradeList = Array.isArray(gradesData) ? gradesData : [];
  const enrolledCourseIds = new Set(enrollmentList.map((e) => e.courseId));

  const filteredCourses = courses.filter((course) => {
    if (tabFilter === 'active') return enrolledCourseIds.has(course.id) && course.isPublished;
    if (tabFilter === 'archive') return !course.isPublished;
    return true;
  });

  const getGrade = (courseId: string) => gradeList.find((g) => g.courseId === courseId);

  const getLetterColor = (letter: string) => {
    if (letter === 'A' || letter === 'A+' || letter === 'A-') return 'var(--forest-600)';
    if (letter === 'B' || letter === 'B+' || letter === 'B-') return '#2980b9';
    if (letter === 'C' || letter === 'C+' || letter === 'C-') return 'var(--muted)';
    return 'var(--warm)';
  };

  return (
    <div className="space-y-5 max-w-3xl mx-auto">
      <div>
        <h1 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
          Courses
        </h1>
        <p className="text-sm mt-1" style={{ color: 'var(--muted)' }}>
          Your enrolled and available courses
        </p>
      </div>

      {/* Segment tabs */}
      <div
        className="flex gap-1 p-1 rounded-xl w-fit"
        style={{ background: 'var(--paper)', border: '1px solid var(--border)' }}
      >
        {(['all', 'active', 'archive'] as const).map((tab) => {
          const isActive = tabFilter === tab;
          const label = tab.charAt(0).toUpperCase() + tab.slice(1);
          return (
            <button
              key={tab}
              onClick={() => setTabFilter(tab)}
              className="px-4 py-1.5 rounded-lg text-sm font-medium capitalize transition-all"
              style={{
                background: isActive ? '#fff' : 'transparent',
                color: isActive ? 'var(--forest-700)' : 'var(--muted)',
                boxShadow: isActive ? 'var(--shadow-sm)' : 'none',
                border: isActive ? '1px solid var(--border)' : '1px solid transparent',
              }}
            >
              {label}
            </button>
          );
        })}
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2
            style={{ width: 28, height: 28, color: 'var(--muted)' }}
            className="animate-spin"
          />
        </div>
      ) : isError ? (
        <div
          className="rounded-xl p-12 text-center"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <p className="text-sm" style={{ color: 'var(--warm)' }}>
            Failed to load courses. Please try again.
          </p>
        </div>
      ) : filteredCourses.length > 0 ? (
        <div className="space-y-3">
          {filteredCourses.map((course) => {
            const isEnrolled = enrolledCourseIds.has(course.id);
            const grade = isEnrolled ? getGrade(course.id) : null;
            const progressPct =
              grade && grade.totalAssignments > 0
                ? Math.round((grade.completedAssignments / grade.totalAssignments) * 100)
                : 0;

            return (
              <div
                key={course.id}
                className="rounded-xl overflow-hidden"
                style={{ background: '#fff', border: '1px solid var(--border)' }}
              >
                <div className="flex items-center gap-4 p-5">
                  <div
                    className="flex items-center justify-center rounded-xl shrink-0 font-bold text-sm"
                    style={{
                      width: 48,
                      height: 48,
                      background: 'var(--forest-700)',
                      color: '#fff',
                    }}
                  >
                    {(course.metadata?.courseCode ?? course.title.substring(0, 2))
                      .substring(0, 2)
                      .toUpperCase()}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-3">
                      <Link
                        href={`/courses/${course.id}`}
                        className="hover:underline decoration-[var(--forest-400)] underline-offset-2"
                      >
                        <h3 className="text-sm font-semibold" style={{ color: 'var(--ink)' }}>
                          {course.title}
                        </h3>
                      </Link>
                      {grade && (
                        <span
                          className="shrink-0 text-base font-bold mono"
                          style={{ color: getLetterColor(grade.letterGrade) }}
                        >
                          {grade.letterGrade}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2 mt-0.5">
                      {course.metadata?.courseCode && (
                        <span
                          className="text-xs font-semibold px-1.5 py-0.5 rounded"
                          style={{ background: 'var(--forest-50)', color: 'var(--forest-700)' }}
                        >
                          {course.metadata.courseCode}
                        </span>
                      )}
                      <span className="text-xs" style={{ color: 'var(--muted)' }}>
                        {course.metadata?.department ?? course.term}
                      </span>
                      {grade && (
                        <span className="text-xs" style={{ color: 'var(--muted)' }}>
                          · {grade.completedAssignments}/{grade.totalAssignments} graded
                        </span>
                      )}
                    </div>
                    {grade && grade.totalAssignments > 0 && (
                      <div className="mt-2.5">
                        <div
                          className="h-1.5 w-full rounded-full overflow-hidden"
                          style={{ background: 'var(--forest-50)' }}
                        >
                          <div
                            className="h-1.5 rounded-full transition-all"
                            style={{
                              width: `${progressPct}%`,
                              background: 'var(--forest-600)',
                            }}
                          />
                        </div>
                      </div>
                    )}
                  </div>
                  {isEnrolled ? (
                    <Link
                      href={`/courses/${course.id}`}
                      className="shrink-0 flex items-center justify-center rounded-lg"
                      style={{
                        width: 32,
                        height: 32,
                        background: 'var(--forest-50)',
                        border: '1px solid var(--forest-200)',
                      }}
                    >
                      <ChevronRight style={{ width: 14, height: 14, color: 'var(--forest-600)' }} />
                    </Link>
                  ) : (
                    <button
                      onClick={() => enrollMutation.mutate(course.id)}
                      disabled={enrollMutation.isPending}
                      className="btn-primary shrink-0 disabled:opacity-50 flex items-center gap-1"
                      style={{ fontSize: 12, padding: '6px 12px' }}
                    >
                      {enrollMutation.isPending ? (
                        <Loader2 style={{ width: 11, height: 11 }} className="animate-spin" />
                      ) : null}
                      Enroll
                    </button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <div
          className="flex flex-col items-center py-16 text-center rounded-xl"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <BookOpen style={{ width: 40, height: 40, color: 'var(--muted)', marginBottom: 12 }} />
          <h3 className="text-base font-semibold" style={{ color: 'var(--ink)' }}>
            No courses found
          </h3>
          <p className="mt-1 text-sm" style={{ color: 'var(--muted)' }}>
            {tabFilter !== 'all'
              ? 'Try switching to All.'
              : 'No courses are available at this time.'}
          </p>
        </div>
      )}
    </div>
  );
}
