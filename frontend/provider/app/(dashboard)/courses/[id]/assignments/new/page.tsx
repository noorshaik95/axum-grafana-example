'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { ArrowLeft, Loader2 } from 'lucide-react'
import Link from 'next/link'
import { useCreateAssignment, useInstructorCourse } from '@/lib/api/hooks'

export default function CreateAssignmentPage() {
  const params = useParams()
  const router = useRouter()
  const courseId = params.id as string
  const { data: course } = useInstructorCourse(courseId)
  const createAssignment = useCreateAssignment()

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [maxPoints, setMaxPoints] = useState('100')
  const [dueDate, setDueDate] = useState('')
  const [latePenalty, setLatePenalty] = useState('0')
  const [maxLateDays, setMaxLateDays] = useState('0')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim() || !dueDate) return
    try {
      await createAssignment.mutateAsync({
        courseId,
        title: title.trim(),
        description: description.trim() || undefined,
        maxPoints: parseInt(maxPoints, 10),
        dueDate: new Date(dueDate).toISOString(),
        latePenaltyPercent: parseFloat(latePenalty),
        maxLateDays: parseInt(maxLateDays, 10),
      })
      router.push(`/courses/${courseId}/assignments`)
    } catch {
      // error state available via createAssignment.error
    }
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
          <Link href={`/courses/${courseId}/assignments`}>
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back to Assignments
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">Create Assignment</h1>
        <p className="text-slate-500 mt-1">{course?.title}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Assignment Details</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <Label htmlFor="title">Title *</Label>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Homework 1: Introduction"
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
                placeholder="Assignment instructions..."
                rows={4}
                className="mt-1.5"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="maxPoints">Max Points</Label>
                <Input
                  id="maxPoints"
                  type="number"
                  value={maxPoints}
                  onChange={(e) => setMaxPoints(e.target.value)}
                  min="0"
                  className="mt-1.5"
                />
              </div>
              <div>
                <Label htmlFor="dueDate">Due Date *</Label>
                <Input
                  id="dueDate"
                  type="datetime-local"
                  value={dueDate}
                  onChange={(e) => setDueDate(e.target.value)}
                  required
                  className="mt-1.5"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="latePenalty">Late Penalty (%/day)</Label>
                <Input
                  id="latePenalty"
                  type="number"
                  value={latePenalty}
                  onChange={(e) => setLatePenalty(e.target.value)}
                  min="0"
                  max="100"
                  className="mt-1.5"
                />
              </div>
              <div>
                <Label htmlFor="maxLateDays">Max Late Days</Label>
                <Input
                  id="maxLateDays"
                  type="number"
                  value={maxLateDays}
                  onChange={(e) => setMaxLateDays(e.target.value)}
                  min="0"
                  className="mt-1.5"
                />
              </div>
            </div>

            {createAssignment.error && (
              <p className="text-sm text-red-600">
                {createAssignment.error instanceof Error
                  ? createAssignment.error.message
                  : 'Failed to create assignment'}
              </p>
            )}

            <div className="flex gap-3 pt-2">
              <Button
                type="submit"
                disabled={createAssignment.isPending || !title.trim() || !dueDate}
              >
                {createAssignment.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {createAssignment.isPending ? 'Creating...' : 'Create Assignment'}
              </Button>
              <Button type="button" variant="outline" asChild>
                <Link href={`/courses/${courseId}/assignments`}>Cancel</Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
