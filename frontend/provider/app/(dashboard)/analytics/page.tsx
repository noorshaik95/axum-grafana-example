'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Loader2, BarChart2, BookOpen, Users, TrendingUp } from 'lucide-react'
import Link from 'next/link'
import { useCourses, useProfile } from '../../../../shared/lib/api/hooks'
import type { Course } from '../../../../shared/lib/api/types'

export default function AnalyticsPage() {
  const { data: profile } = useProfile()
  const {
    data: coursesData,
    isLoading,
    error,
  } = useCourses(profile ? { instructorId: profile.id } : undefined)

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
        Failed to load analytics data. Please try again later.
      </div>
    )
  }

  const publishedCount = courses.filter((c) => c.isPublished).length

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Analytics</h1>
        <p className="text-slate-500 mt-1">Track course metrics and student performance.</p>
      </div>

      {/* Overview Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Total Courses</CardTitle>
            <BookOpen className="h-4 w-4 text-indigo-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{courses.length}</div>
            <p className="text-xs text-slate-500 mt-1">{publishedCount} published</p>
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

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Avg Class Score</CardTitle>
            <TrendingUp className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">--</div>
            <p className="text-xs text-slate-500 mt-1">Across all assignments</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Engagement</CardTitle>
            <BarChart2 className="h-4 w-4 text-amber-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">--</div>
            <p className="text-xs text-slate-500 mt-1">Activity data coming soon</p>
          </CardContent>
        </Card>
      </div>

      {/* Course-level analytics */}
      {courses.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <BarChart2 className="h-12 w-12 text-slate-300 mb-3" />
            <p className="text-slate-500 mb-4">
              No courses yet. Analytics will appear once you create courses.
            </p>
            <Button asChild>
              <Link href="/courses">Go to Courses</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          <h2 className="text-lg font-semibold text-slate-900">Course Overview</h2>
          <div className="grid gap-4">
            {courses.map((course) => (
              <Card key={course.id} className="hover:shadow-sm transition-shadow">
                <CardContent className="flex items-center justify-between p-5">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3 mb-1">
                      <h3 className="font-semibold text-slate-900 truncate">{course.title}</h3>
                      <Badge
                        variant={course.isPublished ? 'default' : 'secondary'}
                        className="shrink-0"
                      >
                        {course.isPublished ? 'Published' : 'Draft'}
                      </Badge>
                    </div>
                    <div className="flex items-center gap-4 text-sm text-slate-500">
                      <span>{course.term}</span>
                      {course.metadata?.courseCode && <span>{course.metadata.courseCode}</span>}
                      {course.metadata?.department && <span>{course.metadata.department}</span>}
                    </div>
                  </div>
                  <Button size="sm" variant="outline" asChild>
                    <Link href={`/courses/${course.id}?tab=grades`}>View Gradebook</Link>
                  </Button>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* Placeholder */}
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12 text-center">
          <BarChart2 className="h-10 w-10 text-slate-300 mb-3" />
          <p className="text-slate-500">
            Detailed analytics charts and engagement trends will be available soon.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
