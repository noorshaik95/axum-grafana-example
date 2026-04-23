'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import {
  Users,
  FileText,
  Loader2,
  AlertCircle,
  ChevronRight,
  CheckCircle2,
  ClipboardList,
  BookOpen,
  ArrowLeft,
} from 'lucide-react';
import { useCourse, useMyEnrollments, useEnrollInCourse } from '../../../../shared/lib/api/hooks';
import { useCourseModules } from '@/lib/api/hooks';
import { useStudentProfile } from '@/lib/api/profile';

export default function CourseDetailPage() {
  const params = useParams();
  const courseId = params.id as string;

  const { data: course, isLoading, isError } = useCourse(courseId);
  const { data: profile } = useStudentProfile();
  const studentId = profile?.id ?? '';
  const { data: enrollments } = useMyEnrollments(studentId);
  const enrollMutation = useEnrollInCourse();
  const { data: modules, isLoading: modulesLoading } = useCourseModules(courseId);

  const enrollmentList = Array.isArray(enrollments) ? enrollments : [];
  const isEnrolled = enrollmentList.some((e) => e.courseId === courseId);
  const moduleList = Array.isArray(modules) ? modules : [];

  // Find first incomplete module
  const nextModule =
    moduleList.find((mod) => {
      const total = mod.lessons?.length ?? 0;
      const done = mod.lessons?.filter((l) => l.completed).length ?? 0;
      return done < total;
    }) ?? moduleList[0];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2
          style={{ width: 28, height: 28, color: 'var(--muted)' }}
          className="animate-spin"
        />
      </div>
    );
  }

  if (isError || !course) {
    return (
      <div className="flex flex-col items-center py-20 text-center">
        <AlertCircle style={{ width: 40, height: 40, color: 'var(--warm)', marginBottom: 12 }} />
        <h2 className="text-lg font-semibold" style={{ color: 'var(--ink)' }}>
          Course not found
        </h2>
        <p className="mt-1 text-sm" style={{ color: 'var(--muted)' }}>
          The course you&apos;re looking for doesn&apos;t exist or could not be loaded.
        </p>
        <Link href="/courses" className="btn-secondary mt-4">
          Back to courses
        </Link>
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto space-y-5">
      <Link
        href="/courses"
        className="inline-flex items-center gap-1.5 text-sm"
        style={{ color: 'var(--muted)' }}
      >
        <ArrowLeft style={{ width: 14, height: 14 }} />
        Courses
      </Link>

      {/* Course header */}
      <div
        className="rounded-xl p-6"
        style={{ background: '#fff', border: '1px solid var(--border)' }}
      >
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <span
                className="text-xs font-bold px-2.5 py-1 rounded"
                style={{ background: 'var(--forest-700)', color: '#fff' }}
              >
                {course.metadata?.courseCode ?? course.metadata?.department ?? 'COURSE'}
              </span>
              <span
                className="text-xs px-2 py-0.5 rounded-full"
                style={{
                  background: 'var(--paper)',
                  color: 'var(--muted)',
                  border: '1px solid var(--border)',
                }}
              >
                {course.term}
              </span>
            </div>
            <h1 className="serif text-2xl sm:text-3xl leading-snug" style={{ color: 'var(--ink)' }}>
              {course.title}
            </h1>
            {course.description && (
              <p className="text-sm" style={{ color: 'var(--muted)' }}>
                {course.description}
              </p>
            )}
          </div>
          <div className="shrink-0">
            {isEnrolled ? (
              <span
                className="inline-flex items-center text-xs font-semibold px-3 py-1.5 rounded-full"
                style={{
                  background: 'var(--forest-50)',
                  color: 'var(--forest-700)',
                  border: '1px solid var(--forest-200)',
                }}
              >
                Enrolled
              </span>
            ) : (
              <button
                onClick={() => enrollMutation.mutate(courseId)}
                disabled={enrollMutation.isPending}
                className="btn-primary flex items-center gap-2 disabled:opacity-50"
                style={{ fontSize: 13 }}
              >
                {enrollMutation.isPending && (
                  <Loader2 style={{ width: 12, height: 12 }} className="animate-spin" />
                )}
                Enroll
              </button>
            )}
          </div>
        </div>

        {/* Stats */}
        <div
          className="mt-5 flex items-center gap-6 pt-4 border-t flex-wrap"
          style={{ borderColor: 'var(--border)' }}
        >
          <div className="flex items-center gap-1.5 text-sm" style={{ color: 'var(--muted)' }}>
            <Users style={{ width: 14, height: 14 }} />
            {course.metadata?.maxStudents ?? '--'} max
          </div>
          <div className="flex items-center gap-1.5 text-sm" style={{ color: 'var(--muted)' }}>
            <BookOpen style={{ width: 14, height: 14 }} />
            {course.metadata?.credits ?? '--'} credits
          </div>
          <div className="flex items-center gap-1.5 text-sm" style={{ color: 'var(--muted)' }}>
            <FileText style={{ width: 14, height: 14 }} />
            {course.metadata?.department ?? '--'}
          </div>
        </div>
      </div>

      {/* Action hero — Next up */}
      {isEnrolled && nextModule && (
        <div
          className="rounded-xl p-6"
          style={{
            background: 'linear-gradient(135deg, var(--forest-700) 0%, var(--forest-900) 100%)',
            color: '#e9efe9',
          }}
        >
          <p className="mono text-xs font-semibold mb-2" style={{ color: 'var(--forest-300)' }}>
            Next up
          </p>
          <h3 className="serif text-xl mb-1" style={{ color: '#f2f7f3' }}>
            {nextModule.name}
          </h3>
          <p className="text-sm mb-4" style={{ color: 'var(--forest-300)' }}>
            {nextModule.lessons?.length ?? 0} lessons
          </p>
          <Link
            href={`/courses/${courseId}/modules/${nextModule.id}`}
            className="btn-amber inline-flex items-center gap-2"
            style={{ fontSize: 13 }}
          >
            Resume at 4:22 →
          </Link>
        </div>
      )}

      <div className="grid gap-5 lg:grid-cols-3">
        {/* Course Modules */}
        <div
          className="lg:col-span-2 rounded-xl overflow-hidden"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <div
            className="flex items-center justify-between p-5 border-b"
            style={{ borderColor: 'var(--border)' }}
          >
            <h2 className="text-sm font-semibold" style={{ color: 'var(--ink)' }}>
              Course Modules
            </h2>
            <span className="text-xs mono" style={{ color: 'var(--muted)' }}>
              {moduleList.length} modules
            </span>
          </div>
          {modulesLoading ? (
            <div className="p-5 space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div
                  key={i}
                  className="h-14 rounded-xl animate-pulse"
                  style={{ background: 'var(--paper)' }}
                />
              ))}
            </div>
          ) : moduleList.length > 0 ? (
            <div className="divide-y" style={{ borderColor: 'var(--border)' }}>
              {moduleList.map((mod) => {
                const totalLessons = mod.lessons?.length ?? 0;
                const completedLessons = mod.lessons?.filter((l) => l.completed).length ?? 0;
                const allComplete = totalLessons > 0 && completedLessons === totalLessons;
                return (
                  <Link
                    key={mod.id}
                    href={`/courses/${courseId}/modules/${mod.id}`}
                    className="flex items-center gap-4 p-4 transition-colors hover:bg-[var(--forest-50)]"
                  >
                    <div
                      className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl"
                      style={{
                        background: allComplete ? 'var(--forest-50)' : 'var(--paper)',
                        border: '1px solid var(--border)',
                      }}
                    >
                      {allComplete ? (
                        <CheckCircle2
                          style={{ width: 18, height: 18, color: 'var(--forest-500)' }}
                        />
                      ) : (
                        <span
                          className="text-sm font-bold mono"
                          style={{ color: 'var(--forest-600)' }}
                        >
                          {mod.displayOrder}
                        </span>
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate" style={{ color: 'var(--ink)' }}>
                        {mod.name}
                      </p>
                      <div className="flex items-center gap-3 mt-1">
                        <span className="text-xs" style={{ color: 'var(--muted)' }}>
                          {totalLessons} lesson{totalLessons !== 1 ? 's' : ''}
                        </span>
                        {totalLessons > 0 && (
                          <>
                            <span className="text-xs" style={{ color: 'var(--muted)' }}>
                              {completedLessons}/{totalLessons}
                            </span>
                            <div
                              className="h-1.5 rounded-full overflow-hidden"
                              style={{ width: 48, background: 'var(--forest-100)' }}
                            >
                              <div
                                className="h-1.5 rounded-full"
                                style={{
                                  width: `${(completedLessons / totalLessons) * 100}%`,
                                  background: 'var(--forest-600)',
                                }}
                              />
                            </div>
                          </>
                        )}
                      </div>
                    </div>
                    <ChevronRight style={{ width: 14, height: 14, color: 'var(--muted)' }} />
                  </Link>
                );
              })}
            </div>
          ) : (
            <div className="p-8 text-center">
              <BookOpen
                style={{ width: 28, height: 28, color: 'var(--muted)', margin: '0 auto 8px' }}
              />
              <p className="text-sm" style={{ color: 'var(--muted)' }}>
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
            className="flex items-center gap-3 p-4 rounded-xl transition-colors hover:bg-[var(--forest-50)]"
            style={{ background: '#fff', border: '1px solid var(--border)' }}
          >
            <div
              className="flex h-10 w-10 items-center justify-center rounded-xl"
              style={{
                background: 'rgba(255,182,72,0.15)',
                border: '1px solid rgba(255,182,72,0.3)',
              }}
            >
              <ClipboardList style={{ width: 18, height: 18, color: 'var(--warm)' }} />
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium" style={{ color: 'var(--ink)' }}>
                Assignments
              </p>
              <p className="text-xs" style={{ color: 'var(--muted)' }}>
                View and submit
              </p>
            </div>
            <ChevronRight style={{ width: 14, height: 14, color: 'var(--muted)' }} />
          </Link>

          {/* Discussion thread */}
          <Link
            href="/discussion/1"
            className="flex items-center gap-3 p-4 rounded-xl transition-colors hover:bg-[var(--forest-50)]"
            style={{ background: '#fff', border: '1px solid var(--border)' }}
          >
            <div
              className="flex h-10 w-10 items-center justify-center rounded-xl"
              style={{ background: 'var(--forest-50)', border: '1px solid var(--forest-200)' }}
            >
              <FileText style={{ width: 18, height: 18, color: 'var(--forest-600)' }} />
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium" style={{ color: 'var(--ink)' }}>
                Discussions
              </p>
              <p className="text-xs" style={{ color: 'var(--muted)' }}>
                3 active threads
              </p>
            </div>
            <ChevronRight style={{ width: 14, height: 14, color: 'var(--muted)' }} />
          </Link>

          {/* Course info */}
          {(course.metadata?.department || course.metadata?.credits) && (
            <div
              className="rounded-xl p-5"
              style={{ background: '#fff', border: '1px solid var(--border)' }}
            >
              <h3
                className="text-xs font-semibold uppercase tracking-wider mb-3"
                style={{ color: 'var(--muted)' }}
              >
                Course info
              </h3>
              <dl className="space-y-2">
                {course.metadata?.department && (
                  <div>
                    <dt className="text-xs" style={{ color: 'var(--muted)' }}>
                      Dept
                    </dt>
                    <dd className="text-sm" style={{ color: 'var(--ink)' }}>
                      {course.metadata.department}
                    </dd>
                  </div>
                )}
                {course.metadata?.credits && (
                  <div>
                    <dt className="text-xs" style={{ color: 'var(--muted)' }}>
                      Credits
                    </dt>
                    <dd className="text-sm" style={{ color: 'var(--ink)' }}>
                      {course.metadata.credits}
                    </dd>
                  </div>
                )}
              </dl>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
