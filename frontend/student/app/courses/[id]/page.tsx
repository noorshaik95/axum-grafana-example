'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import {
  BookOpen,
  Users,
  FileText,
  Loader2,
  AlertCircle,
  ChevronRight,
  CheckCircle2,
  Circle,
  ClipboardList,
} from 'lucide-react';
import {
  useCourse,
  useProfile,
  useMyEnrollments,
  useEnrollInCourse,
} from '../../../../shared/lib/api/hooks';
import { useCourseModules } from '@/lib/api/hooks';

export default function CourseDetailPage() {
  const params = useParams();
  const courseId = params.id as string;

  const { data: course, isLoading, isError } = useCourse(courseId);
  const { data: profile } = useProfile();
  const studentId = profile?.id ?? '';
  const { data: enrollments } = useMyEnrollments(studentId);
  const enrollMutation = useEnrollInCourse();
  const { data: modules, isLoading: modulesLoading } = useCourseModules(courseId);

  const enrollmentList = Array.isArray(enrollments) ? enrollments : [];
  const isEnrolled = enrollmentList.some((e) => e.courseId === courseId);
  const moduleList = Array.isArray(modules) ? modules : [];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--color-text-muted)]" />
      </div>
    );
  }

  if (isError || !course) {
    return (
      <div className="flex flex-col items-center py-20 text-center">
        <AlertCircle className="h-12 w-12 text-[var(--color-error)] mb-3" />
        <h2 className="text-lg font-semibold text-[var(--color-text)]">Course not found</h2>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          The course you&apos;re looking for doesn&apos;t exist or could not be loaded.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Course header */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <span className="rounded-md bg-indigo-600 px-3 py-1 text-sm font-semibold text-white">
                {course.metadata?.courseCode ?? course.metadata?.department ?? 'COURSE'}
              </span>
              <span className="rounded-full border border-[var(--color-border)] px-2.5 py-0.5 text-xs text-[var(--color-text-muted)]">
                {course.term}
              </span>
              {course.isPublished && (
                <span className="rounded-full border border-green-200 bg-green-50 px-2.5 py-0.5 text-xs text-green-700">
                  Published
                </span>
              )}
            </div>
            <h1 className="text-xl font-bold text-[var(--color-text)]">{course.title}</h1>
            {course.description && (
              <p className="text-sm text-[var(--color-text-muted)]">{course.description}</p>
            )}
          </div>
          <div className="shrink-0">
            {isEnrolled ? (
              <span className="inline-flex items-center rounded-lg bg-green-50 px-4 py-2 text-sm font-medium text-green-700">
                Enrolled
              </span>
            ) : (
              <button
                onClick={() => enrollMutation.mutate(courseId)}
                disabled={enrollMutation.isPending}
                className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 transition-colors"
              >
                {enrollMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                Enroll in Course
              </button>
            )}
          </div>
        </div>

        {/* Stats */}
        <div className="mt-5 grid grid-cols-2 gap-4 border-t border-[var(--color-border)] pt-4 sm:grid-cols-3">
          <div className="flex items-center gap-2">
            <Users className="h-4 w-4 text-[var(--color-text-muted)]" />
            <span className="text-sm text-[var(--color-text-muted)]">
              {course.metadata?.maxStudents ?? '--'} Max Students
            </span>
          </div>
          <div className="flex items-center gap-2">
            <BookOpen className="h-4 w-4 text-[var(--color-text-muted)]" />
            <span className="text-sm text-[var(--color-text-muted)]">
              {course.metadata?.credits ?? '--'} Credits
            </span>
          </div>
          <div className="flex items-center gap-2">
            <FileText className="h-4 w-4 text-[var(--color-text-muted)]" />
            <span className="text-sm text-[var(--color-text-muted)]">
              {course.metadata?.department ?? '--'}
            </span>
          </div>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Course Modules */}
        <div className="lg:col-span-2 rounded-xl border border-[var(--color-border)] bg-white shadow-sm">
          <div className="flex items-center justify-between p-5 border-b border-[var(--color-border)]">
            <h2 className="text-base font-semibold text-[var(--color-text)]">Course Modules</h2>
            <span className="text-xs text-[var(--color-text-muted)]">
              {moduleList.length} modules
            </span>
          </div>
          {modulesLoading ? (
            <div className="p-5 space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-16 rounded-lg bg-gray-100 animate-pulse" />
              ))}
            </div>
          ) : moduleList.length > 0 ? (
            <div className="divide-y divide-[var(--color-border)]">
              {moduleList.map((mod) => {
                const totalLessons = mod.lessons?.length ?? 0;
                const completedLessons = mod.lessons?.filter((l) => l.completed).length ?? 0;
                const allComplete = totalLessons > 0 && completedLessons === totalLessons;
                return (
                  <Link
                    key={mod.id}
                    href={`/courses/${courseId}/modules/${mod.id}`}
                    className="flex items-center gap-4 p-4 hover:bg-[var(--color-bg-muted)] transition-colors"
                  >
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-indigo-50">
                      {allComplete ? (
                        <CheckCircle2 className="h-5 w-5 text-green-600" />
                      ) : (
                        <span className="text-sm font-semibold text-indigo-600">
                          {mod.displayOrder}
                        </span>
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-[var(--color-text)] truncate">
                        {mod.name}
                      </p>
                      <div className="flex items-center gap-3 mt-1">
                        <span className="text-xs text-[var(--color-text-muted)]">
                          {totalLessons} lesson{totalLessons !== 1 ? 's' : ''}
                        </span>
                        {totalLessons > 0 && (
                          <>
                            <span className="text-xs text-[var(--color-text-muted)]">
                              {completedLessons}/{totalLessons} complete
                            </span>
                            <div className="h-1.5 w-16 rounded-full bg-gray-100">
                              <div
                                className="h-1.5 rounded-full bg-indigo-600 transition-all"
                                style={{
                                  width: `${(completedLessons / totalLessons) * 100}%`,
                                }}
                              />
                            </div>
                          </>
                        )}
                      </div>
                    </div>
                    <ChevronRight className="h-4 w-4 text-[var(--color-text-muted)]" />
                  </Link>
                );
              })}
            </div>
          ) : (
            <div className="p-8 text-center">
              <BookOpen className="mx-auto h-8 w-8 text-[var(--color-text-muted)]" />
              <p className="mt-2 text-sm text-[var(--color-text-muted)]">
                No modules available yet.
              </p>
            </div>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-4">
          {/* Assignments link */}
          <Link
            href={`/courses/${courseId}/assignments`}
            className="flex items-center gap-3 rounded-xl border border-[var(--color-border)] bg-white p-4 shadow-sm hover:bg-[var(--color-bg-muted)] transition-colors"
          >
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-50 text-amber-600">
              <ClipboardList className="h-5 w-5" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-[var(--color-text)]">Assignments</p>
              <p className="text-xs text-[var(--color-text-muted)]">View and submit assignments</p>
            </div>
            <ChevronRight className="h-4 w-4 text-[var(--color-text-muted)]" />
          </Link>

          {/* Syllabus */}
          {course.syllabus && (
            <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
              <h3 className="text-sm font-semibold text-[var(--color-text)] mb-3">Syllabus</h3>
              <a
                href={course.syllabus}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 text-sm text-indigo-600 hover:text-indigo-700"
              >
                <FileText className="h-4 w-4" />
                View Syllabus
              </a>
            </div>
          )}

          {/* Course Info */}
          <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
            <h3 className="text-sm font-semibold text-[var(--color-text)] mb-3">
              Course Information
            </h3>
            <dl className="space-y-2">
              <div>
                <dt className="text-xs text-[var(--color-text-muted)]">Term</dt>
                <dd className="text-sm text-[var(--color-text)]">{course.term}</dd>
              </div>
              {course.metadata?.department && (
                <div>
                  <dt className="text-xs text-[var(--color-text-muted)]">Department</dt>
                  <dd className="text-sm text-[var(--color-text)]">{course.metadata.department}</dd>
                </div>
              )}
              {course.metadata?.credits && (
                <div>
                  <dt className="text-xs text-[var(--color-text-muted)]">Credits</dt>
                  <dd className="text-sm text-[var(--color-text)]">{course.metadata.credits}</dd>
                </div>
              )}
              {course.metadata?.tags && course.metadata.tags.length > 0 && (
                <div>
                  <dt className="text-xs text-[var(--color-text-muted)]">Tags</dt>
                  <dd className="mt-1 flex flex-wrap gap-1">
                    {course.metadata.tags.map((tag) => (
                      <span
                        key={tag}
                        className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600"
                      >
                        {tag}
                      </span>
                    ))}
                  </dd>
                </div>
              )}
            </dl>
          </div>
        </div>
      </div>
    </div>
  );
}
