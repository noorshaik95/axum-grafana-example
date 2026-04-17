'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Loader2, Calendar as CalendarIcon, BookOpen, FileText } from 'lucide-react'
import Link from 'next/link'
import { useCourses, useProfile } from '../../../../shared/lib/api/hooks'
import type { Course } from '../../../../shared/lib/api/types'

export default function CalendarPage() {
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
        Failed to load calendar data. Please try again later.
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Calendar</h1>
        <p className="text-slate-500 mt-1">View your teaching schedule and upcoming deadlines.</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle>Schedule</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="rounded-lg bg-slate-50 p-12 text-center">
                <CalendarIcon className="h-16 w-16 mx-auto mb-4 text-slate-300" />
                <h3 className="text-lg font-semibold text-slate-900 mb-2">Calendar View</h3>
                <p className="text-slate-500 text-sm">
                  Interactive calendar will be available in a future update.
                </p>
              </div>
            </CardContent>
          </Card>
        </div>

        <div>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <BookOpen className="h-5 w-5" />
                Your Courses
              </CardTitle>
            </CardHeader>
            <CardContent>
              {courses.length === 0 ? (
                <p className="text-sm text-slate-500 text-center py-4">No courses found.</p>
              ) : (
                <div className="space-y-3">
                  {courses.map((course) => (
                    <Link
                      key={course.id}
                      href={`/courses/${course.id}`}
                      className="block rounded-lg border p-3 hover:bg-slate-50 transition-colors"
                    >
                      <div className="flex items-start justify-between">
                        <div className="min-w-0 flex-1">
                          <h4 className="text-sm font-semibold text-slate-900 truncate">
                            {course.title}
                          </h4>
                          <p className="text-xs text-slate-500 mt-0.5">{course.term}</p>
                        </div>
                        <Badge
                          variant={course.isPublished ? 'default' : 'secondary'}
                          className="text-xs ml-2 shrink-0"
                        >
                          {course.isPublished ? 'Active' : 'Draft'}
                        </Badge>
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          <Card className="mt-4">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <FileText className="h-5 w-5" />
                Upcoming Deadlines
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-slate-500 text-center py-4">
                Assignment deadline tracking will be available soon.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
