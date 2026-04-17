'use client'

import { useParams } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { ArrowLeft, Loader2 } from 'lucide-react'
import Link from 'next/link'
import { useInstructorCourse } from '@/lib/api/hooks'
import { ContentManager } from '@/components/courses/content/ContentManager'

export default function ContentPage() {
  const params = useParams()
  const courseId = params.id as string
  const { data: course, isLoading } = useInstructorCourse(courseId)

  if (isLoading) {
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
          <Link href={`/courses/${courseId}`}>
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back to {course?.title ?? 'Course'}
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">Content Manager</h1>
        <p className="text-slate-500 mt-1">
          Organize modules and lessons with drag-and-drop reordering.
        </p>
      </div>

      <ContentManager courseId={courseId} />
    </div>
  )
}
