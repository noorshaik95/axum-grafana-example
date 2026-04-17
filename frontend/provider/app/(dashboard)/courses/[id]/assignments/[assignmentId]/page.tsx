'use client'

import { useParams } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ArrowLeft, Loader2, Users, FileText, Clock } from 'lucide-react'
import Link from 'next/link'
import { useAssignment, useSubmissions, useInstructorCourse } from '@/lib/api/hooks'

export default function AssignmentDetailPage() {
  const params = useParams()
  const courseId = params.id as string
  const assignmentId = params.assignmentId as string
  const { data: course } = useInstructorCourse(courseId)
  const { data: assignment, isLoading } = useAssignment(assignmentId)
  const { data: submissions } = useSubmissions(assignmentId)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    )
  }

  if (!assignment) {
    return (
      <div className="space-y-4">
        <Button variant="outline" asChild>
          <Link href={`/courses/${courseId}/assignments`}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Assignments
          </Link>
        </Button>
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-red-700">
          Assignment not found.
        </div>
      </div>
    )
  }

  const gradedCount = submissions?.filter((s) => s.status === 'graded').length ?? 0
  const pendingCount = submissions?.filter((s) => s.status === 'submitted').length ?? 0
  const isPastDue = new Date(assignment.dueDate) < new Date()

  return (
    <div className="space-y-6">
      <div>
        <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
          <Link href={`/courses/${courseId}/assignments`}>
            <ArrowLeft className="h-4 w-4 mr-1" />
            {course?.title ?? 'Back'}
          </Link>
        </Button>
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900">{assignment.title}</h1>
            <div className="flex items-center gap-3 mt-2">
              <Badge variant={isPastDue ? 'secondary' : 'default'}>
                {isPastDue ? 'Past Due' : 'Active'}
              </Badge>
              <span className="text-sm text-slate-500">{assignment.maxPoints} points</span>
              <span className="text-sm text-slate-500">
                Due: {new Date(assignment.dueDate).toLocaleDateString()}
              </span>
            </div>
          </div>
          <Button asChild>
            <Link href={`/courses/${courseId}/assignments/${assignmentId}/submissions`}>
              Grade Submissions
            </Link>
          </Button>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="flex items-center gap-3 pt-6">
            <Users className="h-8 w-8 text-indigo-500" />
            <div>
              <p className="text-2xl font-bold">{submissions?.length ?? 0}</p>
              <p className="text-sm text-slate-500">Total Submissions</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 pt-6">
            <FileText className="h-8 w-8 text-amber-500" />
            <div>
              <p className="text-2xl font-bold">{pendingCount}</p>
              <p className="text-sm text-slate-500">Pending Grading</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 pt-6">
            <Clock className="h-8 w-8 text-emerald-500" />
            <div>
              <p className="text-2xl font-bold">{gradedCount}</p>
              <p className="text-sm text-slate-500">Graded</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {assignment.description && (
        <Card>
          <CardHeader>
            <CardTitle>Description</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-slate-600 whitespace-pre-wrap">{assignment.description}</p>
          </CardContent>
        </Card>
      )}

      {(assignment.latePenaltyPercent > 0 || assignment.maxLateDays > 0) && (
        <Card>
          <CardHeader>
            <CardTitle>Late Policy</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-slate-600">
            <p>Penalty: {assignment.latePenaltyPercent}% per day</p>
            <p>Maximum late days: {assignment.maxLateDays}</p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
