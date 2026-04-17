'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Loader2, BookOpen, FileText, ClipboardCheck, Settings2 } from 'lucide-react'
import Link from 'next/link'
import { useProfile } from '../../../../shared/lib/api/hooks'
import { useInstructorCourses, usePendingSubmissions, useAssignments } from '@/lib/api/hooks'
import { formatDateTime } from '@/lib/utils'
import type { Course, Assignment } from '../../../../shared/lib/api/types'

function CourseGradingCard({ course }: { course: Course }) {
  const { data: assignments, isLoading } = useAssignments(course.id)

  const upcoming = (assignments ?? []).filter((a) => new Date(a.dueDate) >= new Date())
  const past = (assignments ?? []).filter((a) => new Date(a.dueDate) < new Date())

  return (
    <Card className="hover:shadow-md transition-shadow">
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
        {isLoading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
          </div>
        ) : !assignments || assignments.length === 0 ? (
          <p className="text-sm text-slate-500">No assignments yet.</p>
        ) : (
          <div className="space-y-3">
            <div className="grid grid-cols-3 gap-3 text-center">
              <div className="rounded-lg bg-slate-50 p-2">
                <p className="text-lg font-bold text-slate-900">{assignments.length}</p>
                <p className="text-xs text-slate-500">Total</p>
              </div>
              <div className="rounded-lg bg-amber-50 p-2">
                <p className="text-lg font-bold text-amber-700">{upcoming.length}</p>
                <p className="text-xs text-slate-500">Upcoming</p>
              </div>
              <div className="rounded-lg bg-emerald-50 p-2">
                <p className="text-lg font-bold text-emerald-700">{past.length}</p>
                <p className="text-xs text-slate-500">Past Due</p>
              </div>
            </div>

            <div className="space-y-2">
              {assignments.slice(0, 3).map((assignment) => {
                const isPastDue = new Date(assignment.dueDate) < new Date()
                return (
                  <div
                    key={assignment.id}
                    className="flex items-center justify-between rounded-lg border p-2.5"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <FileText className="h-4 w-4 text-slate-400 shrink-0" />
                      <div className="min-w-0">
                        <p className="text-sm font-medium truncate">{assignment.title}</p>
                        <p className="text-xs text-slate-500">
                          {assignment.maxPoints} pts &middot; Due{' '}
                          {new Date(assignment.dueDate).toLocaleDateString()}
                        </p>
                      </div>
                    </div>
                    {isPastDue && (
                      <Badge variant="secondary" className="text-xs shrink-0 ml-2">
                        Past Due
                      </Badge>
                    )}
                  </div>
                )
              })}
            </div>

            <Button size="sm" variant="outline" className="w-full" asChild>
              <Link href={`/courses/${course.id}/assignments`}>View Assignments</Link>
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default function GradingPage() {
  const { data: profile } = useProfile()
  const {
    data: coursesData,
    isLoading,
    error,
  } = useInstructorCourses(profile ? { instructorId: profile.id } : undefined)
  const { data: pendingSubmissions } = usePendingSubmissions()

  const courses: readonly Course[] = coursesData?.data ?? []

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-red-700">
        Failed to load grading data. Please try again later.
      </div>
    )
  }

  const publishedCourses = courses.filter((c) => c.isPublished)
  const draftCourses = courses.filter((c) => !c.isPublished)
  const pendingCount = pendingSubmissions?.length ?? 0

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Grading</h1>
          <p className="text-slate-500 mt-1">Grade student assignments and manage grading rules.</p>
        </div>
        <Button variant="outline" asChild>
          <Link href="/grading/rules">
            <Settings2 className="h-4 w-4 mr-2" />
            Grading Rules
          </Link>
        </Button>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">
              Pending Submissions
            </CardTitle>
            <ClipboardCheck className="h-4 w-4 text-amber-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{pendingCount}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Courses</CardTitle>
            <BookOpen className="h-4 w-4 text-indigo-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{courses.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Published</CardTitle>
            <FileText className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{publishedCourses.length}</div>
          </CardContent>
        </Card>
      </div>

      {/* Pending submissions quick view */}
      {pendingSubmissions && pendingSubmissions.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Recent Pending Submissions</CardTitle>
          </CardHeader>
          <CardContent>
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
                    <p className="text-xs text-slate-500">{formatDateTime(sub.submittedAt)}</p>
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
          </CardContent>
        </Card>
      )}

      {courses.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <ClipboardCheck className="h-12 w-12 text-slate-300 mb-3" />
            <p className="text-slate-500 mb-4">
              No courses found. Create a course to start grading.
            </p>
            <Button asChild>
              <Link href="/courses">Go to Courses</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Tabs defaultValue="all">
          <TabsList>
            <TabsTrigger value="all">All Courses ({courses.length})</TabsTrigger>
            <TabsTrigger value="published">Published ({publishedCourses.length})</TabsTrigger>
            {draftCourses.length > 0 && (
              <TabsTrigger value="drafts">Drafts ({draftCourses.length})</TabsTrigger>
            )}
          </TabsList>

          <TabsContent value="all" className="mt-4">
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {courses.map((course) => (
                <CourseGradingCard key={course.id} course={course} />
              ))}
            </div>
          </TabsContent>

          <TabsContent value="published" className="mt-4">
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {publishedCourses.map((course) => (
                <CourseGradingCard key={course.id} course={course} />
              ))}
            </div>
          </TabsContent>

          {draftCourses.length > 0 && (
            <TabsContent value="drafts" className="mt-4">
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {draftCourses.map((course) => (
                  <CourseGradingCard key={course.id} course={course} />
                ))}
              </div>
            </TabsContent>
          )}
        </Tabs>
      )}
    </div>
  )
}
