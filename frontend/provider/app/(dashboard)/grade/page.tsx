'use client'

import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { ArrowRight, ClipboardList } from 'lucide-react'
import { Spinner } from '../../../../shared/components/ui'
import { EmptyStateIllustrated } from '../../../../shared/components'
import { listCourses } from '../../../lib/api/courses'
import * as gradingApi from '../../../lib/api/grading'
import type { Assignment, Course } from '../../../../shared/lib/api/types'

interface AssignmentRow {
  assignment: Assignment
  courseTitle: string
}

export default function GradeIndexPage() {
  const coursesQuery = useQuery({
    queryKey: ['provider-grade-courses'],
    queryFn: () => listCourses({ pageSize: 50 }),
  })

  const courses: readonly Course[] = coursesQuery.data?.data ?? []

  const assignmentQueries = useQuery({
    queryKey: ['provider-grade-all-assignments', courses.map((c) => c.id).join(',')],
    queryFn: async () => {
      const rows: AssignmentRow[] = []
      await Promise.all(
        courses.map(async (course) => {
          try {
            const assignments = await gradingApi.listAssignments(course.id)
            for (const a of assignments) {
              rows.push({ assignment: a, courseTitle: course.title })
            }
          } catch {
            // Swallow per-course failures so the page still renders other courses.
          }
        })
      )
      rows.sort(
        (a, b) =>
          new Date(a.assignment.dueDate).getTime() - new Date(b.assignment.dueDate).getTime()
      )
      return rows
    },
    enabled: courses.length > 0,
  })

  if (coursesQuery.isLoading || assignmentQueries.isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Spinner />
      </div>
    )
  }

  const rows = assignmentQueries.data ?? []

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-serif text-3xl text-warm-900">Grade</h1>
        <p className="mt-1 text-sm text-warm-700">
          Open an assignment to review its pattern-grouped grading queue.
        </p>
      </div>

      {rows.length === 0 ? (
        <EmptyStateIllustrated
          illustration="all-done"
          title="No assignments to grade"
          description="Once students submit work, their assignments will appear here."
        />
      ) : (
        <ul className="divide-y divide-warm-200 overflow-hidden rounded-card border border-warm-200 bg-paper">
          {rows.map(({ assignment, courseTitle }) => (
            <li key={assignment.id}>
              <Link
                href={`/grade/${assignment.id}`}
                className="flex items-center gap-4 px-5 py-4 transition-colors hover:bg-warm-50"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-forest-100 text-forest-900">
                  <ClipboardList className="h-5 w-5" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="truncate font-serif text-base text-warm-900">
                    {assignment.title}
                  </div>
                  <div className="mt-0.5 flex items-center gap-2 text-xs text-warm-700">
                    <span className="truncate">{courseTitle}</span>
                    <span className="opacity-40">·</span>
                    <span>Due {new Date(assignment.dueDate).toLocaleDateString()}</span>
                    <span className="opacity-40">·</span>
                    <span>{assignment.maxPoints} pts</span>
                  </div>
                </div>
                <ArrowRight className="h-4 w-4 shrink-0 text-warm-500" />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
