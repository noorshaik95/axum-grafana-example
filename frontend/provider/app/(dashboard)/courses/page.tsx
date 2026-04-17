'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Plus, Loader2, BookOpen, Search } from 'lucide-react'
import Link from 'next/link'
import { useState, useMemo } from 'react'
import { useInstructorCourses } from '@/lib/api/hooks'
import { useProfile } from '../../../../shared/lib/api/hooks'
import type { Course } from '../../../../shared/lib/api/types'

export default function CoursesPage() {
  const { data: profile } = useProfile()
  const {
    data: coursesData,
    isLoading,
    error,
  } = useInstructorCourses(profile ? { instructorId: profile.id } : undefined)
  const [search, setSearch] = useState('')

  const courses: readonly Course[] = coursesData?.data ?? []

  const filtered = useMemo(() => {
    if (!search) return courses
    const q = search.toLowerCase()
    return courses.filter(
      (c) =>
        c.title.toLowerCase().includes(q) ||
        c.description?.toLowerCase().includes(q) ||
        c.metadata?.courseCode?.toLowerCase().includes(q)
    )
  }, [courses, search])

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
        Failed to load courses. Please try again later.
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">My Courses</h1>
          <p className="text-slate-500 mt-1">Manage and create courses</p>
        </div>
        <Button asChild>
          <Link href="/courses/new">
            <Plus className="h-4 w-4 mr-2" />
            Create Course
          </Link>
        </Button>
      </div>

      {courses.length > 0 && (
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search courses..."
            className="pl-9"
          />
        </div>
      )}

      {filtered.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <BookOpen className="h-12 w-12 text-slate-300 mb-3" />
            <p className="text-slate-500 mb-4">
              {search
                ? 'No courses match your search.'
                : 'No courses found. Create your first course to get started.'}
            </p>
            {!search && (
              <Button asChild>
                <Link href="/courses/new">
                  <Plus className="h-4 w-4 mr-2" />
                  Create Course
                </Link>
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((course) => (
            <Card key={course.id} className="hover:shadow-md transition-shadow">
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
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
                  {course.description || 'No description provided.'}
                </p>
                {course.metadata?.courseCode && (
                  <p className="text-xs text-slate-400 mb-3">Code: {course.metadata.courseCode}</p>
                )}
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
  )
}
