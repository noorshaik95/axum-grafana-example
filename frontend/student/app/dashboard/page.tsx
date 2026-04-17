'use client';

import Link from 'next/link';
import {
  BookOpen,
  FileText,
  GraduationCap,
  Flame,
  ArrowRight,
  Loader2,
  Clock,
  Video,
  BarChart3,
  Calendar,
} from 'lucide-react';
import { useProfile, useMyEnrollments, useCourses } from '../../../shared/lib/api/hooks';
import {
  useStudentAssignments,
  useGradesOverview,
  useVideoSessions,
  useStudentProgress,
} from '@/lib/api/hooks';

function formatDueDate(dateStr: string): string {
  const now = new Date();
  const due = new Date(dateStr);
  const diffMs = due.getTime() - now.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));

  if (diffMs < 0) return 'Overdue';
  if (diffHours < 1) return 'Due soon';
  if (diffHours < 24) return `${diffHours}h left`;
  if (diffDays === 1) return 'Tomorrow';
  return `${diffDays} days`;
}

function dueUrgencyClass(dateStr: string): string {
  const diffMs = new Date(dateStr).getTime() - Date.now();
  const diffDays = diffMs / (1000 * 60 * 60 * 24);
  if (diffMs < 0) return 'text-red-600 font-semibold';
  if (diffDays <= 1) return 'text-red-500 font-medium';
  if (diffDays <= 3) return 'text-amber-500 font-medium';
  return 'text-[var(--color-text-muted)]';
}

export default function DashboardPage() {
  const { data: profile, isLoading: profileLoading } = useProfile();
  const studentId = profile?.id ?? '';
  const { data: enrollments, isLoading: enrollmentsLoading } = useMyEnrollments(studentId);
  const { data: coursesData } = useCourses();
  const { data: upcomingAssignments } = useStudentAssignments({ upcoming: true });
  const { data: gradesOverview } = useGradesOverview();
  const { data: videoSessions } = useVideoSessions({ upcoming: true });
  const { data: progress } = useStudentProgress();

  const enrollmentList = Array.isArray(enrollments) ? enrollments : [];
  const courses = coursesData?.data ?? [];
  const assignments = Array.isArray(upcomingAssignments) ? upcomingAssignments.slice(0, 5) : [];
  const grades = Array.isArray(gradesOverview) ? gradesOverview : [];
  const sessions = Array.isArray(videoSessions) ? videoSessions : [];
  const firstName = profile?.firstName ?? 'Student';

  const todaySessions = sessions.filter((s) => {
    const start = new Date(s.startTime);
    const now = new Date();
    return start.toDateString() === now.toDateString() || s.isActive;
  });

  const gpa =
    progress?.gpaEstimate ??
    (grades.length > 0
      ? (grades.reduce((sum, g) => sum + g.currentGrade, 0) / grades.length / 25).toFixed(1)
      : '--');

  const completionPct = progress?.overallCompletionPercent ?? '--';
  const streakDays = progress?.streakDays ?? '--';

  return (
    <div className="space-y-6">
      {/* Welcome */}
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">
          {profileLoading ? (
            <span className="inline-flex items-center gap-2">
              <Loader2 className="h-5 w-5 animate-spin" />
              Loading...
            </span>
          ) : (
            `Welcome back, ${firstName}!`
          )}
        </h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Here&apos;s what&apos;s happening with your courses today.
        </p>
      </div>

      {/* Quick Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[
          {
            label: 'Enrolled Courses',
            value: enrollmentsLoading ? null : enrollmentList.length,
            icon: BookOpen,
            color: 'bg-indigo-50 text-indigo-600',
          },
          {
            label: 'Upcoming Due',
            value: assignments.length,
            icon: FileText,
            color: 'bg-amber-50 text-amber-600',
          },
          {
            label: 'GPA Estimate',
            value: gpa,
            icon: GraduationCap,
            color: 'bg-green-50 text-green-600',
          },
          {
            label: 'Study Streak',
            value: typeof streakDays === 'number' ? `${streakDays}d` : streakDays,
            icon: Flame,
            color: 'bg-orange-50 text-orange-600',
          },
        ].map((stat) => (
          <div
            key={stat.label}
            className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm"
          >
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium text-[var(--color-text-muted)]">{stat.label}</p>
              <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${stat.color}`}>
                <stat.icon className="h-4 w-4" />
              </div>
            </div>
            <div className="mt-2">
              {stat.value === null ? (
                <Loader2 className="h-6 w-6 animate-spin text-[var(--color-text-muted)]" />
              ) : (
                <p className="text-2xl font-bold text-[var(--color-text)]">{stat.value}</p>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Completion + Streak bar */}
      {typeof completionPct === 'number' && (
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between mb-2">
            <p className="text-sm font-medium text-[var(--color-text)]">Overall Completion</p>
            <span className="text-sm font-semibold text-indigo-600">{completionPct}%</span>
          </div>
          <div className="h-2 w-full rounded-full bg-gray-100">
            <div
              className="h-2 rounded-full bg-indigo-600 transition-all"
              style={{ width: `${completionPct}%` }}
            />
          </div>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* My Courses */}
        <div className="rounded-xl border border-[var(--color-border)] bg-white shadow-sm">
          <div className="flex items-center justify-between p-5 pb-3">
            <h2 className="text-base font-semibold text-[var(--color-text)]">My Courses</h2>
            <Link
              href="/courses"
              className="flex items-center gap-1 text-xs font-medium text-indigo-600 hover:text-indigo-700"
            >
              View all <ArrowRight className="h-3 w-3" />
            </Link>
          </div>
          <div className="px-5 pb-5 space-y-3">
            {enrollmentsLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <div
                  key={i}
                  className="flex items-center gap-4 rounded-lg border border-[var(--color-border)] p-4"
                >
                  <div className="h-10 w-10 animate-pulse rounded-lg bg-gray-200" />
                  <div className="flex-1 space-y-2">
                    <div className="h-4 w-48 animate-pulse rounded bg-gray-200" />
                    <div className="h-3 w-32 animate-pulse rounded bg-gray-200" />
                  </div>
                </div>
              ))
            ) : courses.length > 0 ? (
              courses.slice(0, 4).map((course) => (
                <Link
                  key={course.id}
                  href={`/courses/${course.id}`}
                  className="flex items-center gap-4 rounded-lg border border-[var(--color-border)] p-4 hover:bg-[var(--color-bg-muted)] transition-colors"
                >
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-indigo-100 text-xs font-bold text-indigo-600">
                    {(course.metadata?.courseCode ?? course.title.substring(0, 2)).toUpperCase()}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-[var(--color-text)] truncate">
                      {course.title}
                    </p>
                    <p className="text-xs text-[var(--color-text-muted)]">
                      {course.metadata?.department ?? course.term}
                    </p>
                  </div>
                  <ArrowRight className="h-4 w-4 text-[var(--color-text-muted)]" />
                </Link>
              ))
            ) : (
              <div className="py-8 text-center">
                <BookOpen className="mx-auto h-8 w-8 text-[var(--color-text-muted)]" />
                <p className="mt-2 text-sm text-[var(--color-text-muted)]">No courses yet.</p>
                <Link
                  href="/courses"
                  className="mt-3 inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors"
                >
                  Browse Courses
                  <ArrowRight className="h-4 w-4" />
                </Link>
              </div>
            )}
          </div>
        </div>

        {/* Upcoming Deadlines */}
        <div className="rounded-xl border border-[var(--color-border)] bg-white shadow-sm">
          <div className="flex items-center justify-between p-5 pb-3">
            <h2 className="text-base font-semibold text-[var(--color-text)]">Upcoming Deadlines</h2>
            <Link
              href="/assignments"
              className="flex items-center gap-1 text-xs font-medium text-indigo-600 hover:text-indigo-700"
            >
              View all <ArrowRight className="h-3 w-3" />
            </Link>
          </div>
          <div className="px-5 pb-5 space-y-2">
            {assignments.length > 0 ? (
              assignments.map((a) => (
                <Link
                  key={a.id}
                  href={`/courses/${a.courseId}/assignments/${a.id}`}
                  className="flex items-center justify-between rounded-lg border border-[var(--color-border)] p-3 hover:bg-[var(--color-bg-muted)] transition-colors"
                >
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium text-[var(--color-text)] truncate">
                      {a.title}
                    </p>
                    <p className="text-xs text-[var(--color-text-muted)]">{a.courseTitle}</p>
                  </div>
                  <div className="ml-3 flex items-center gap-1.5">
                    <Clock className="h-3.5 w-3.5 text-[var(--color-text-muted)]" />
                    <span className={`text-xs ${dueUrgencyClass(a.dueDate)}`}>
                      {formatDueDate(a.dueDate)}
                    </span>
                  </div>
                </Link>
              ))
            ) : (
              <div className="py-8 text-center">
                <Calendar className="mx-auto h-8 w-8 text-[var(--color-text-muted)]" />
                <p className="mt-2 text-sm text-[var(--color-text-muted)]">
                  No upcoming deadlines.
                </p>
              </div>
            )}
          </div>
        </div>

        {/* Recent Grades */}
        <div className="rounded-xl border border-[var(--color-border)] bg-white shadow-sm">
          <div className="flex items-center justify-between p-5 pb-3">
            <h2 className="text-base font-semibold text-[var(--color-text)]">Course Grades</h2>
            <Link
              href="/grades"
              className="flex items-center gap-1 text-xs font-medium text-indigo-600 hover:text-indigo-700"
            >
              View all <ArrowRight className="h-3 w-3" />
            </Link>
          </div>
          <div className="px-5 pb-5 space-y-2">
            {grades.length > 0 ? (
              grades.slice(0, 5).map((g) => (
                <Link
                  key={g.courseId}
                  href={`/grades/${g.courseId}`}
                  className="flex items-center justify-between rounded-lg border border-[var(--color-border)] p-3 hover:bg-[var(--color-bg-muted)] transition-colors"
                >
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium text-[var(--color-text)] truncate">
                      {g.courseTitle}
                    </p>
                    <p className="text-xs text-[var(--color-text-muted)]">
                      {g.completedAssignments}/{g.totalAssignments} graded
                    </p>
                  </div>
                  <div className="ml-3 flex items-center gap-2">
                    <span className="text-sm font-semibold text-[var(--color-text)]">
                      {g.letterGrade}
                    </span>
                    <span className="text-xs text-[var(--color-text-muted)]">
                      {Math.round(g.currentGrade)}%
                    </span>
                  </div>
                </Link>
              ))
            ) : (
              <div className="py-8 text-center">
                <BarChart3 className="mx-auto h-8 w-8 text-[var(--color-text-muted)]" />
                <p className="mt-2 text-sm text-[var(--color-text-muted)]">No grades yet.</p>
              </div>
            )}
          </div>
        </div>

        {/* Today's Live Classes */}
        <div className="rounded-xl border border-[var(--color-border)] bg-white shadow-sm">
          <div className="flex items-center justify-between p-5 pb-3">
            <h2 className="text-base font-semibold text-[var(--color-text)]">
              Today&apos;s Live Classes
            </h2>
            <Link
              href="/video"
              className="flex items-center gap-1 text-xs font-medium text-indigo-600 hover:text-indigo-700"
            >
              View all <ArrowRight className="h-3 w-3" />
            </Link>
          </div>
          <div className="px-5 pb-5 space-y-2">
            {todaySessions.length > 0 ? (
              todaySessions.map((s) => (
                <Link
                  key={s.id}
                  href={`/video/${s.id}`}
                  className="flex items-center justify-between rounded-lg border border-[var(--color-border)] p-3 hover:bg-[var(--color-bg-muted)] transition-colors"
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium text-[var(--color-text)] truncate">
                        {s.title}
                      </p>
                      {s.isActive && (
                        <span className="flex h-2 w-2 rounded-full bg-red-500 animate-pulse" />
                      )}
                    </div>
                    <p className="text-xs text-[var(--color-text-muted)]">
                      {s.courseTitle} -- {s.hostName}
                    </p>
                  </div>
                  <div className="ml-3 text-xs text-[var(--color-text-muted)]">
                    {new Date(s.startTime).toLocaleTimeString([], {
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </div>
                </Link>
              ))
            ) : (
              <div className="py-8 text-center">
                <Video className="mx-auto h-8 w-8 text-[var(--color-text-muted)]" />
                <p className="mt-2 text-sm text-[var(--color-text-muted)]">
                  No live classes today.
                </p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Quick actions */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {[
          { label: 'Browse Courses', href: '/courses', icon: BookOpen },
          { label: 'View Assignments', href: '/assignments', icon: FileText },
          { label: 'Check Grades', href: '/grades', icon: GraduationCap },
          { label: 'Live Classes', href: '/video', icon: Video },
        ].map((action) => (
          <Link
            key={action.label}
            href={action.href}
            className="flex items-center gap-3 rounded-xl border border-[var(--color-border)] bg-white p-4 shadow-sm hover:bg-[var(--color-bg-muted)] transition-colors"
          >
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600">
              <action.icon className="h-4 w-4" />
            </div>
            <span className="text-sm font-medium text-[var(--color-text)]">{action.label}</span>
          </Link>
        ))}
      </div>
    </div>
  );
}
