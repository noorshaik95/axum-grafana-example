'use client'

import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { ArrowLeft, Loader2, Trash2 } from 'lucide-react'
import Link from 'next/link'
import { useInstructorCourse, useUpdateCourse, useDeleteCourse } from '@/lib/api/hooks'

export default function CourseSettingsPage() {
  const params = useParams()
  const router = useRouter()
  const courseId = params.id as string
  const { data: course, isLoading } = useInstructorCourse(courseId)
  const updateCourse = useUpdateCourse()
  const deleteCourse = useDeleteCourse()

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [term, setTerm] = useState('')
  const [courseCode, setCourseCode] = useState('')
  const [maxStudents, setMaxStudents] = useState('')
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  useEffect(() => {
    if (course) {
      setTitle(course.title)
      setDescription(course.description ?? '')
      setTerm(course.term)
      setCourseCode(course.metadata?.courseCode ?? '')
      setMaxStudents(course.metadata?.maxStudents?.toString() ?? '')
    }
  }, [course])

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    await updateCourse.mutateAsync({
      id: courseId,
      data: {
        title,
        description,
        term,
        metadata: {
          ...course?.metadata,
          courseCode: courseCode || undefined,
          maxStudents: maxStudents ? parseInt(maxStudents, 10) : undefined,
        },
      },
    })
  }

  async function handlePublishToggle() {
    await updateCourse.mutateAsync({
      id: courseId,
      data: { isPublished: !course?.isPublished },
    })
  }

  async function handleDelete() {
    await deleteCourse.mutateAsync(courseId)
    router.push('/courses')
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
          <Link href={`/courses/${courseId}`}>
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back to Course
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">Course Settings</h1>
        <p className="text-slate-500 mt-1">{course?.title}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>General</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSave} className="space-y-5">
            <div>
              <Label htmlFor="title">Course Title</Label>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="mt-1.5"
              />
            </div>
            <div>
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={4}
                className="mt-1.5"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="term">Term</Label>
                <Input
                  id="term"
                  value={term}
                  onChange={(e) => setTerm(e.target.value)}
                  className="mt-1.5"
                />
              </div>
              <div>
                <Label htmlFor="courseCode">Course Code</Label>
                <Input
                  id="courseCode"
                  value={courseCode}
                  onChange={(e) => setCourseCode(e.target.value)}
                  className="mt-1.5"
                />
              </div>
            </div>
            <div>
              <Label htmlFor="maxStudents">Max Students</Label>
              <Input
                id="maxStudents"
                type="number"
                value={maxStudents}
                onChange={(e) => setMaxStudents(e.target.value)}
                className="mt-1.5"
              />
            </div>
            {updateCourse.error && <p className="text-sm text-red-600">Failed to save changes.</p>}
            <Button type="submit" disabled={updateCourse.isPending}>
              {updateCourse.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Publish Status</CardTitle>
        </CardHeader>
        <CardContent className="flex items-center justify-between">
          <div>
            <Badge variant={course?.isPublished ? 'default' : 'secondary'}>
              {course?.isPublished ? 'Published' : 'Draft'}
            </Badge>
            <p className="text-sm text-slate-500 mt-1">
              {course?.isPublished
                ? 'Students can view this course.'
                : 'Only you can view this course.'}
            </p>
          </div>
          <Button variant="outline" onClick={handlePublishToggle} disabled={updateCourse.isPending}>
            {course?.isPublished ? 'Unpublish' : 'Publish'}
          </Button>
        </CardContent>
      </Card>

      <Card className="border-red-200">
        <CardHeader>
          <CardTitle className="text-red-700">Danger Zone</CardTitle>
        </CardHeader>
        <CardContent>
          {showDeleteConfirm ? (
            <div className="space-y-3">
              <p className="text-sm text-red-700">
                Are you sure? This action cannot be undone. All course data will be permanently
                deleted.
              </p>
              <div className="flex gap-2">
                <Button
                  variant="destructive"
                  onClick={handleDelete}
                  disabled={deleteCourse.isPending}
                >
                  {deleteCourse.isPending ? 'Deleting...' : 'Confirm Delete'}
                </Button>
                <Button variant="outline" onClick={() => setShowDeleteConfirm(false)}>
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-slate-900">Delete this course</p>
                <p className="text-sm text-slate-500">Once deleted, this cannot be recovered.</p>
              </div>
              <Button variant="destructive" onClick={() => setShowDeleteConfirm(true)}>
                <Trash2 className="h-4 w-4 mr-2" />
                Delete Course
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
