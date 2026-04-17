'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { ArrowLeft, Loader2 } from 'lucide-react'
import Link from 'next/link'
import { useCreateCourse } from '@/lib/api/hooks'
import { useProfile } from '../../../../../shared/lib/api/hooks'

export default function CreateCoursePage() {
  const router = useRouter()
  const { data: profile } = useProfile()
  const createCourse = useCreateCourse()

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [term, setTerm] = useState('Spring 2026')
  const [courseCode, setCourseCode] = useState('')
  const [department, setDepartment] = useState('')
  const [maxStudents, setMaxStudents] = useState('')
  const [credits, setCredits] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim() || !profile) return
    try {
      const course = await createCourse.mutateAsync({
        title: title.trim(),
        description: description.trim(),
        term,
        instructorId: profile.id,
        coInstructorIds: [],
        isPublished: false,
        prerequisiteCourseIds: [],
        metadata: {
          courseCode: courseCode || undefined,
          department: department || undefined,
          maxStudents: maxStudents ? parseInt(maxStudents, 10) : undefined,
          credits: credits ? parseInt(credits, 10) : undefined,
        },
      })
      router.push(`/courses/${course.id}`)
    } catch {
      // error state available via createCourse.error
    }
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
          <Link href="/courses">
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back to Courses
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">Create New Course</h1>
        <p className="text-slate-500 mt-1">Fill in the details to create a new course.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Course Details</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <Label htmlFor="title">Course Title *</Label>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Introduction to Computer Science"
                required
                className="mt-1.5"
              />
            </div>

            <div>
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="A brief description of the course..."
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
                  placeholder="Spring 2026"
                  className="mt-1.5"
                />
              </div>
              <div>
                <Label htmlFor="courseCode">Course Code</Label>
                <Input
                  id="courseCode"
                  value={courseCode}
                  onChange={(e) => setCourseCode(e.target.value)}
                  placeholder="CS101"
                  className="mt-1.5"
                />
              </div>
            </div>

            <div className="grid grid-cols-3 gap-4">
              <div>
                <Label htmlFor="department">Department</Label>
                <Input
                  id="department"
                  value={department}
                  onChange={(e) => setDepartment(e.target.value)}
                  placeholder="Computer Science"
                  className="mt-1.5"
                />
              </div>
              <div>
                <Label htmlFor="maxStudents">Max Students</Label>
                <Input
                  id="maxStudents"
                  type="number"
                  value={maxStudents}
                  onChange={(e) => setMaxStudents(e.target.value)}
                  placeholder="30"
                  className="mt-1.5"
                />
              </div>
              <div>
                <Label htmlFor="credits">Credits</Label>
                <Input
                  id="credits"
                  type="number"
                  value={credits}
                  onChange={(e) => setCredits(e.target.value)}
                  placeholder="3"
                  className="mt-1.5"
                />
              </div>
            </div>

            {createCourse.error && (
              <p className="text-sm text-red-600">
                {createCourse.error instanceof Error
                  ? createCourse.error.message
                  : 'Failed to create course'}
              </p>
            )}

            <div className="flex gap-3 pt-2">
              <Button type="submit" disabled={createCourse.isPending || !title.trim()}>
                {createCourse.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {createCourse.isPending ? 'Creating...' : 'Create Course'}
              </Button>
              <Button type="button" variant="outline" asChild>
                <Link href="/courses">Cancel</Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
