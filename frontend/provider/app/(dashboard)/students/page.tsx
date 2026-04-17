'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Loader2, Users, BookOpen } from 'lucide-react'
import Link from 'next/link'
import { useCourses, useProfile } from '../../../../shared/lib/api/hooks'
import type { Course } from '../../../../shared/lib/api/types'

export default function StudentsPage() {
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
        Failed to load data. Please try again later.
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Students</h1>
        <p className="text-slate-500 mt-1">View students enrolled in your courses.</p>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Your Courses</CardTitle>
            <BookOpen className="h-4 w-4 text-indigo-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{courses.length}</div>
            <p className="text-xs text-slate-500 mt-1">
              {courses.filter((c) => c.isPublished).length} published
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-slate-500">Total Students</CardTitle>
            <Users className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">--</div>
            <p className="text-xs text-slate-500 mt-1">Enrollment data coming soon</p>
          </CardContent>
        </Card>
      </div>

      {/* Course list */}
      {courses.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <Users className="h-12 w-12 text-slate-300 mb-3" />
            <p className="text-slate-500 mb-4">
              No courses found. Create a course and enroll students to see them here.
            </p>
            <Button asChild>
              <Link href="/courses">Go to Courses</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          <h2 className="text-lg font-semibold text-slate-900">Students by Course</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {courses.map((course) => (
              <Card key={course.id} className="hover:shadow-md transition-shadow">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="min-w-0 flex-1">
                      <CardTitle className="text-base truncate">{course.title}</CardTitle>
                      <p className="text-sm text-slate-500 mt-1">{course.term}</p>
                    </div>
                    <Badge
                      variant={course.isPublished ? 'default' : 'secondary'}
                      className="ml-2 shrink-0"
                    >
                      {course.isPublished ? 'Published' : 'Draft'}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-slate-600 line-clamp-2 mb-4">
                    {course.description || 'No description.'}
                  </p>
                  {course.metadata?.maxStudents && (
                    <p className="text-xs text-slate-400 mb-3">
                      Max capacity: {course.metadata.maxStudents}
                    </p>
                  )}
                  <Button size="sm" variant="outline" className="w-full" asChild>
                    <Link href={`/courses/${course.id}`}>View Course & Students</Link>
                  </Button>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
