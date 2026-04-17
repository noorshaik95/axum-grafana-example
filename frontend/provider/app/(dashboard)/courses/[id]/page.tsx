'use client'

import { useParams } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Loader2,
  ArrowLeft,
  FileText,
  Users,
  Calendar,
  BookOpen,
  Settings,
  GripVertical,
  ClipboardCheck,
} from 'lucide-react'
import Link from 'next/link'
import { useInstructorCourse, useAssignments } from '@/lib/api/hooks'

export default function CourseDetailPage() {
  const params = useParams()
  const courseId = params.id as string
  const { data: course, isLoading, error } = useInstructorCourse(courseId)
  const { data: assignments, isLoading: assignmentsLoading } = useAssignments(courseId)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    )
  }

  if (error || !course) {
    return (
      <div className="space-y-4">
        <Button variant="outline" asChild>
          <Link href="/courses">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Courses
          </Link>
        </Button>
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-red-700">
          Failed to load course.
        </div>
      </div>
    )
  }

  const quickLinks = [
    {
      name: 'Content',
      href: `/courses/${courseId}/content`,
      icon: GripVertical,
      description: 'Manage modules and lessons',
    },
    {
      name: 'Assignments',
      href: `/courses/${courseId}/assignments`,
      icon: FileText,
      description: `${assignmentsLoading ? '...' : (assignments?.length ?? 0)} assignments`,
    },
    {
      name: 'Students',
      href: `/courses/${courseId}/students`,
      icon: Users,
      description: 'View student progress',
    },
    {
      name: 'Submissions',
      href: `/courses/${courseId}/assignments`,
      icon: ClipboardCheck,
      description: 'Grade student work',
    },
    {
      name: 'Settings',
      href: `/courses/${courseId}/settings`,
      icon: Settings,
      description: 'Course configuration',
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
            <Link href="/courses">
              <ArrowLeft className="h-4 w-4 mr-1" />
              Back
            </Link>
          </Button>
          <h1 className="text-2xl font-bold text-slate-900">{course.title}</h1>
          <div className="flex items-center gap-3 mt-2">
            <Badge variant={course.isPublished ? 'default' : 'secondary'}>
              {course.isPublished ? 'Published' : 'Draft'}
            </Badge>
            <span className="text-sm text-slate-500">{course.term}</span>
            {course.metadata?.courseCode && (
              <span className="text-sm text-slate-500">{course.metadata.courseCode}</span>
            )}
          </div>
        </div>
        <Button variant="outline" asChild>
          <Link href={`/courses/${courseId}/settings`}>
            <Settings className="h-4 w-4 mr-2" />
            Settings
          </Link>
        </Button>
      </div>

      {/* Description */}
      {course.description && (
        <Card>
          <CardHeader>
            <CardTitle>Description</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-slate-600">{course.description}</p>
          </CardContent>
        </Card>
      )}

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="flex items-center gap-3 pt-6">
            <FileText className="h-8 w-8 text-indigo-500" />
            <div>
              <p className="text-2xl font-bold">
                {assignmentsLoading ? '...' : (assignments?.length ?? 0)}
              </p>
              <p className="text-sm text-slate-500">Assignments</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 pt-6">
            <Users className="h-8 w-8 text-emerald-500" />
            <div>
              <p className="text-2xl font-bold">--</p>
              <p className="text-sm text-slate-500">Enrolled Students</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 pt-6">
            <Calendar className="h-8 w-8 text-amber-500" />
            <div>
              <p className="text-2xl font-bold">{course.term}</p>
              <p className="text-sm text-slate-500">Term</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Quick Links */}
      <div>
        <h2 className="text-lg font-semibold text-slate-900 mb-4">Manage</h2>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {quickLinks.map((link) => (
            <Link key={link.name} href={link.href}>
              <Card className="hover:shadow-md hover:border-indigo-200 transition-all cursor-pointer h-full">
                <CardContent className="flex items-center gap-3 p-4">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-50">
                    <link.icon className="h-5 w-5 text-indigo-600" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-slate-900">{link.name}</p>
                    <p className="text-xs text-slate-500">{link.description}</p>
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      </div>

      {/* Recent Assignments */}
      {assignments && assignments.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-slate-900">Recent Assignments</h2>
            <Button size="sm" variant="outline" asChild>
              <Link href={`/courses/${courseId}/assignments`}>View All</Link>
            </Button>
          </div>
          <div className="space-y-2">
            {assignments.slice(0, 5).map((a) => (
              <Card key={a.id}>
                <CardContent className="flex items-center justify-between p-4">
                  <div>
                    <Link
                      href={`/courses/${courseId}/assignments/${a.id}`}
                      className="text-sm font-medium text-slate-900 hover:text-indigo-600"
                    >
                      {a.title}
                    </Link>
                    <p className="text-xs text-slate-500 mt-0.5">
                      {a.maxPoints} pts &middot; Due {new Date(a.dueDate).toLocaleDateString()}
                    </p>
                  </div>
                  <Badge variant={new Date(a.dueDate) < new Date() ? 'secondary' : 'default'}>
                    {new Date(a.dueDate) < new Date() ? 'Past Due' : 'Active'}
                  </Badge>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
