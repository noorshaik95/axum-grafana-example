'use client'

import { useParams } from 'next/navigation'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ArrowLeft, Plus, FileText, Loader2 } from 'lucide-react'
import Link from 'next/link'
import { useAssignments, useInstructorCourse } from '@/lib/api/hooks'

export default function AssignmentsPage() {
  const params = useParams()
  const courseId = params.id as string
  const { data: course } = useInstructorCourse(courseId)
  const { data: assignments, isLoading } = useAssignments(courseId)

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
            <Link href={`/courses/${courseId}`}>
              <ArrowLeft className="h-4 w-4 mr-1" />
              Back to {course?.title ?? 'Course'}
            </Link>
          </Button>
          <h1 className="text-2xl font-bold text-slate-900">Assignments</h1>
          <p className="text-slate-500 mt-1">Manage assignments for this course.</p>
        </div>
        <Button asChild>
          <Link href={`/courses/${courseId}/assignments/new`}>
            <Plus className="h-4 w-4 mr-2" />
            Create Assignment
          </Link>
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
        </div>
      ) : !assignments || assignments.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <FileText className="h-10 w-10 text-slate-300 mb-3" />
            <p className="text-slate-500 mb-4">No assignments yet.</p>
            <Button asChild>
              <Link href={`/courses/${courseId}/assignments/new`}>
                <Plus className="h-4 w-4 mr-2" />
                Create First Assignment
              </Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {assignments.map((assignment) => {
            const isPastDue = new Date(assignment.dueDate) < new Date()
            return (
              <Card key={assignment.id} className="hover:shadow-sm transition-shadow">
                <CardContent className="flex items-center justify-between p-4">
                  <div>
                    <Link
                      href={`/courses/${courseId}/assignments/${assignment.id}`}
                      className="font-medium text-slate-900 hover:text-indigo-600 transition-colors"
                    >
                      {assignment.title}
                    </Link>
                    <div className="flex items-center gap-3 mt-1 text-sm text-slate-500">
                      <span>{assignment.maxPoints} points</span>
                      <span>Due: {new Date(assignment.dueDate).toLocaleDateString()}</span>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {isPastDue && (
                      <Badge variant="secondary" className="text-xs">
                        Past Due
                      </Badge>
                    )}
                    {assignment.latePenaltyPercent > 0 && (
                      <Badge variant="outline" className="text-xs">
                        -{assignment.latePenaltyPercent}%/day
                      </Badge>
                    )}
                    <Button size="sm" variant="outline" asChild>
                      <Link href={`/courses/${courseId}/assignments/${assignment.id}/submissions`}>
                        Submissions
                      </Link>
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
