'use client';

import { useState } from 'react';
import Link from 'next/link';
import { BookOpen, Search, Loader2, Users } from 'lucide-react';
import {
  useCourses,
  useMyEnrollments,
  useProfile,
  useEnrollInCourse,
} from '../../../shared/lib/api/hooks';

type TabFilter = 'all' | 'enrolled' | 'available';

export default function CoursesPage() {
  const [searchQuery, setSearchQuery] = useState('');
  const [tabFilter, setTabFilter] = useState<TabFilter>('all');

  const { data: profile } = useProfile();
  const studentId = profile?.id ?? '';
  const {
    data: coursesData,
    isLoading,
    isError,
  } = useCourses({
    search: searchQuery || undefined,
  });
  const { data: enrollments } = useMyEnrollments(studentId);
  const enrollMutation = useEnrollInCourse();

  const courses = coursesData?.data ?? [];
  const enrollmentList = Array.isArray(enrollments) ? enrollments : [];
  const enrolledCourseIds = new Set(enrollmentList.map((e) => e.courseId));

  const filteredCourses = courses.filter((course) => {
    if (tabFilter === 'enrolled') return enrolledCourseIds.has(course.id);
    if (tabFilter === 'available') return !enrolledCourseIds.has(course.id);
    return true;
  });

  const handleEnroll = (courseId: string) => {
    enrollMutation.mutate(courseId);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">Courses</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Browse and enroll in available courses
        </p>
      </div>

      {/* Search + filter tabs */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--color-text-muted)]" />
          <input
            type="text"
            placeholder="Search courses..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-lg border border-[var(--color-border)] bg-white py-2 pl-10 pr-4 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>
        <div className="flex gap-2">
          {(['all', 'enrolled', 'available'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setTabFilter(tab)}
              className={`rounded-lg px-3 py-2 text-xs font-medium capitalize transition-colors ${
                tabFilter === tab
                  ? 'bg-indigo-600 text-white'
                  : 'bg-white border border-[var(--color-border)] text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)]'
              }`}
            >
              {tab}
            </button>
          ))}
        </div>
      </div>

      {/* Course grid */}
      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm"
            >
              <div className="space-y-3">
                <div className="h-5 w-16 animate-pulse rounded bg-gray-200" />
                <div className="h-4 w-full animate-pulse rounded bg-gray-200" />
                <div className="h-3 w-32 animate-pulse rounded bg-gray-200" />
                <div className="h-8 w-24 animate-pulse rounded bg-gray-200" />
              </div>
            </div>
          ))}
        </div>
      ) : isError ? (
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-12 text-center shadow-sm">
          <p className="text-sm text-[var(--color-error)]">
            Failed to load courses. Please try again.
          </p>
        </div>
      ) : filteredCourses.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filteredCourses.map((course) => {
            const isEnrolled = enrolledCourseIds.has(course.id);
            return (
              <div
                key={course.id}
                className="group rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm hover:shadow-md transition-shadow"
              >
                <div className="flex items-center gap-2 mb-3">
                  <span className="rounded-md bg-indigo-600 px-2.5 py-1 text-xs font-semibold text-white">
                    {course.metadata?.courseCode ?? course.metadata?.department ?? 'COURSE'}
                  </span>
                  {course.isPublished && (
                    <span className="text-xs text-green-600 font-medium">Published</span>
                  )}
                </div>
                <Link href={`/courses/${course.id}`}>
                  <h3 className="text-sm font-semibold text-[var(--color-text)] group-hover:text-indigo-600 transition-colors">
                    {course.title}
                  </h3>
                </Link>
                {course.description && (
                  <p className="mt-1 text-xs text-[var(--color-text-muted)] line-clamp-2">
                    {course.description}
                  </p>
                )}
                <div className="mt-3 flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
                  <Users className="h-3.5 w-3.5" />
                  <span>{course.metadata?.maxStudents ?? '--'} max students</span>
                </div>
                <div className="mt-4">
                  {isEnrolled ? (
                    <Link
                      href={`/courses/${course.id}`}
                      className="inline-flex items-center rounded-lg bg-green-50 px-3 py-1.5 text-xs font-medium text-green-700 hover:bg-green-100 transition-colors"
                    >
                      Enrolled -- Continue
                    </Link>
                  ) : (
                    <button
                      onClick={() => handleEnroll(course.id)}
                      disabled={enrollMutation.isPending}
                      className="inline-flex items-center rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50 transition-colors"
                    >
                      {enrollMutation.isPending ? (
                        <Loader2 className="h-3 w-3 animate-spin mr-1" />
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
        <div className="flex flex-col items-center py-16 text-center">
          <BookOpen className="h-12 w-12 text-[var(--color-text-muted)] mb-3" />
          <h3 className="text-base font-semibold text-[var(--color-text)]">No courses found</h3>
          <p className="mt-1 text-sm text-[var(--color-text-muted)]">
            {searchQuery
              ? 'Try adjusting your search terms.'
              : 'No courses are available at this time.'}
          </p>
        </div>
      )}
    </div>
  );
}
