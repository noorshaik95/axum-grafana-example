'use client'

import { useParams } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { ArrowLeft, Loader2 } from 'lucide-react'
import Link from 'next/link'
import { useAssignment, useSubmissions, useInstructorCourse } from '@/lib/api/hooks'
import { SubmissionGrader } from '@/components/grading/SubmissionGrader'

export default function SubmissionsPage() {
  const params = useParams()
  const courseId = params.id as string
  const assignmentId = params.assignmentId as string
  const { data: course } = useInstructorCourse(courseId)
  const { data: assignment, isLoading: assignmentLoading } = useAssignment(assignmentId)
  const { data: submissions, isLoading: submissionsLoading } = useSubmissions(assignmentId)

  if (assignmentLoading || submissionsLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
          <Link href={`/courses/${courseId}/assignments/${assignmentId}`}>
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back to {assignment?.title ?? 'Assignment'}
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">Grade Submissions</h1>
        <p className="text-slate-500 mt-1">
          {course?.title} &mdash; {assignment?.title}
        </p>
      </div>

      <SubmissionGrader
        assignmentId={assignmentId}
        maxPoints={assignment?.maxPoints ?? 100}
        submissions={submissions ?? []}
      />
    </div>
  )
}
