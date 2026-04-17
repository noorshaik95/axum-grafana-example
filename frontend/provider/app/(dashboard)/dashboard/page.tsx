'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  BookOpen,
  Users,
  ClipboardCheck,
  Video,
  TrendingUp,
  Loader2,
  FileText,
  ArrowRight,
} from 'lucide-react'
import Link from 'next/link'
import { useProfile } from '../../../../shared/lib/api/hooks'
import { useInstructorCourses, usePendingSubmissions, useVideoSessions } from '@/lib/api/hooks'
import { formatDateTime } from '@/lib/utils'
import type { Course } from '../../../../shared/lib/api/types'

export default function DashboardPage() {
  const { data: profile, isLoading: profileLoading } = useProfile()
  const {
    data: coursesData,
    isLoading: coursesLoading,
    error: coursesError,
  } = useInstructorCourses(profile ? { instructorId: profile.id } : undefined)
  const { data: pendingSubmissions, isLoading: submissionsLoading } = usePendingSubmissions()
  const { data: videoSessions, isLoading: sessionsLoading } = useVideoSessions({
    status: 'scheduled',
  })

  const courses: readonly Course[] = coursesData?.data ?? []
  const isLoading = profileLoading || coursesLoading

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    )
  }

  if (coursesError) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-red-700">
        Failed to load dashboard data. Please try again later.
      </div>
    )
  }

  const publishedCount = courses.filter((c) => c.isPublished).length
  const draftCount = courses.filter((c) => !c.isPublished).length
  const pendingCount = pendingSubmissions?.length ?? 0
  const upcomingSessions =
    videoSessions?.filter((s) => {
      const sessionDate = new Date(s.scheduledAt)
      const today = new Date()
      today.setHours(0, 0, 0, 0)
      const tomorrow = new Date(today)
      tomorrow.setDate(tomorrow.getDate() + 1)
      return sessionDate >= today && sessionDate < tomorrow
    }) ?? []

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">
          Welcome back{profile ? `, ${profile.firstName}` : ''}
        </h1>
        <p className="text-slate-500 mt-1">Here is an overview of your teaching activity.</p>
      </div>

      {/* Stat Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Active Courses</CardTitle>
            <BookOpen className="h-4 w-4 text-indigo-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{publishedCount}</div>
            <p className="text-xs text-slate-500 mt-1">
              {draftCount} draft{draftCount !== 1 ? 's' : ''}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Pending Grades</CardTitle>
            <ClipboardCheck className="h-4 w-4 text-amber-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{submissionsLoading ? '...' : pendingCount}</div>
            <p className="text-xs text-slate-500 mt-1">Submissions awaiting review</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Today's Sessions</CardTitle>
            <Video className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">
              {sessionsLoading ? '...' : upcomingSessions.length}
            </div>
            <p className="text-xs text-slate-500 mt-1">Scheduled for today</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Total Students</CardTitle>
            <Users className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">--</div>
            <p className="text-xs text-slate-500 mt-1">Across all courses</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Recent Submissions */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Pending Submissions</CardTitle>
            <Button size="sm" variant="ghost" asChild>
              <Link href="/grading">
                View All
                <ArrowRight className="h-4 w-4 ml-1" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent>
            {submissionsLoading ? (
              <div className="flex items-center justify-center py-6">
                <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
              </div>
            ) : !pendingSubmissions || pendingSubmissions.length === 0 ? (
              <div className="text-center py-6">
                <FileText className="h-8 w-8 text-slate-300 mx-auto mb-2" />
                <p className="text-sm text-slate-500">No pending submissions.</p>
              </div>
            ) : (
              <div className="space-y-2">
                {pendingSubmissions.slice(0, 5).map((sub) => (
                  <div
                    key={sub.id}
                    className="flex items-center justify-between rounded-lg border p-3"
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-slate-900 truncate">
                        Student {sub.studentId.slice(0, 8)}...
                      </p>
                      <p className="text-xs text-slate-500">
                        Submitted {formatDateTime(sub.submittedAt)}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      {sub.isLate && (
                        <Badge variant="secondary" className="text-xs">
                          Late
                        </Badge>
                      )}
                      <Badge variant="outline">{sub.status}</Badge>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Today's Sessions */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Today's Video Sessions</CardTitle>
            <Button size="sm" variant="ghost" asChild>
              <Link href="/video">
                View All
                <ArrowRight className="h-4 w-4 ml-1" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent>
            {sessionsLoading ? (
              <div className="flex items-center justify-center py-6">
                <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
              </div>
            ) : upcomingSessions.length === 0 ? (
              <div className="text-center py-6">
                <Video className="h-8 w-8 text-slate-300 mx-auto mb-2" />
                <p className="text-sm text-slate-500">No sessions today.</p>
              </div>
            ) : (
              <div className="space-y-2">
                {upcomingSessions.map((session) => (
                  <div
                    key={session.id}
                    className="flex items-center justify-between rounded-lg border p-3"
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-slate-900 truncate">{session.title}</p>
                      <p className="text-xs text-slate-500">
                        {formatDateTime(session.scheduledAt)} &middot; {session.duration} min
                      </p>
                    </div>
                    <Badge variant={session.status === 'live' ? 'destructive' : 'default'}>
                      {session.status === 'live' ? 'LIVE' : 'Scheduled'}
                    </Badge>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* My Courses */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-slate-900">My Courses</h2>
          <Button asChild size="sm" variant="outline">
            <Link href="/courses">View All</Link>
          </Button>
        </div>

        {courses.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-12 text-center">
              <BookOpen className="h-12 w-12 text-slate-300 mb-3" />
              <p className="text-slate-500 mb-4">You have not created any courses yet.</p>
              <Button asChild>
                <Link href="/courses/new">Create Your First Course</Link>
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {courses.slice(0, 6).map((course) => (
              <Card key={course.id} className="hover:shadow-md transition-shadow">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div>
                      <CardTitle className="text-base">{course.title}</CardTitle>
                      <p className="text-sm text-slate-500 mt-1">{course.term}</p>
                    </div>
                    <Badge variant={course.isPublished ? 'default' : 'secondary'}>
                      {course.isPublished ? 'Published' : 'Draft'}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-slate-600 line-clamp-2 mb-4">
                    {course.description || 'No description provided.'}
                  </p>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" className="flex-1" asChild>
                      <Link href={`/courses/${course.id}`}>View</Link>
                    </Button>
                    <Button size="sm" className="flex-1" asChild>
                      <Link href={`/courses/${course.id}/assignments`}>Assignments</Link>
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
