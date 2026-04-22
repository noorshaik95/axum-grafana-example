'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '../../../../shared/components/ui/card'
import { Button } from '../../../../shared/components/ui/button'
import { Badge } from '../../../../shared/components/ui/badge'
import { Loader2, Users, BookOpen } from 'lucide-react'
import Link from 'next/link'
import { useInstructorCourses, useCourseRoster } from '@/lib/api/hooks'
import { useProfile } from '../../../../shared/lib/api/hooks'

export default function StudentsPage() {
  const { data: profile } = useProfile()
  const coursesQuery = useInstructorCourses(profile ? { instructorId: profile.id } : undefined)
  const courses = coursesQuery.data?.data ?? []

  const [selectedCourseId, setSelectedCourseId] = useState<string>('')
  const effectiveCourseId = selectedCourseId || courses[0]?.id || ''

  const rosterQuery = useCourseRoster(effectiveCourseId)

  if (coursesQuery.isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    )
  }

  if (coursesQuery.error) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-red-700">
        Failed to load data. Please try again later.
      </div>
    )
  }

  const roster = rosterQuery.data?.enrollments ?? []

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
            <CardTitle className="text-sm font-medium text-slate-500">
              Students in selected course
            </CardTitle>
            <Users className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">
              {rosterQuery.isLoading ? (
                <Loader2 className="h-5 w-5 animate-spin inline" />
              ) : (
                (rosterQuery.data?.total_count ?? roster.length)
              )}
            </div>
            <p className="text-xs text-slate-500 mt-1">
              {courses.find((c) => c.id === effectiveCourseId)?.title ?? '—'}
            </p>
          </CardContent>
        </Card>
      </div>

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
        <>
          {/* Course selector */}
          <div className="flex items-center gap-3">
            <label className="text-sm font-medium text-slate-700">Course:</label>
            <select
              value={effectiveCourseId}
              onChange={(e) => setSelectedCourseId(e.target.value)}
              className="rounded-lg border border-slate-200 px-3 py-2 text-sm bg-white"
            >
              {courses.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.title}
                </option>
              ))}
            </select>
          </div>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Enrolled students</CardTitle>
            </CardHeader>
            <CardContent>
              {rosterQuery.isLoading ? (
                <div className="flex items-center justify-center py-10">
                  <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
                </div>
              ) : rosterQuery.error ? (
                <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
                  Failed to load roster.
                </div>
              ) : roster.length === 0 ? (
                <p className="text-sm text-slate-500 text-center py-8">
                  No students enrolled in this course yet.
                </p>
              ) : (
                <div className="divide-y divide-slate-100">
                  {roster.map((e) => (
                    <div key={e.id} className="flex items-center justify-between py-3">
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 items-center justify-center rounded-full bg-indigo-100 text-xs font-semibold text-indigo-700">
                          {(e.student_name ?? e.student_id).slice(0, 2).toUpperCase()}
                        </div>
                        <div>
                          <p className="text-sm font-medium text-slate-900">
                            {e.student_name ?? e.student_id}
                          </p>
                          <p className="text-xs text-slate-500">
                            Enrolled {new Date(e.enrolled_at).toLocaleDateString()}
                          </p>
                        </div>
                      </div>
                      <Badge variant={e.status === 'active' ? 'default' : 'secondary'}>
                        {e.status}
                      </Badge>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
