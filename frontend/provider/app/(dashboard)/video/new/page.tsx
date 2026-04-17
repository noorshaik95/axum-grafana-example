'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ArrowLeft, Loader2 } from 'lucide-react'
import Link from 'next/link'
import { useCreateVideoSession, useInstructorCourses } from '@/lib/api/hooks'
import { useProfile } from '../../../../../shared/lib/api/hooks'
import type { Course } from '../../../../../shared/lib/api/types'

export default function ScheduleSessionPage() {
  const router = useRouter()
  const { data: profile } = useProfile()
  const { data: coursesData } = useInstructorCourses(
    profile ? { instructorId: profile.id } : undefined
  )
  const createSession = useCreateVideoSession()

  const courses: readonly Course[] = coursesData?.data ?? []

  const [title, setTitle] = useState('')
  const [courseId, setCourseId] = useState('')
  const [scheduledAt, setScheduledAt] = useState('')
  const [duration, setDuration] = useState('60')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim() || !courseId || !scheduledAt) return
    try {
      await createSession.mutateAsync({
        title: title.trim(),
        courseId,
        scheduledAt: new Date(scheduledAt).toISOString(),
        duration: parseInt(duration, 10),
      })
      router.push('/video')
    } catch {
      // error via createSession.error
    }
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
          <Link href="/video">
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back to Sessions
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">Schedule Video Session</h1>
        <p className="text-slate-500 mt-1">Create a new live class session.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Session Details</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <Label htmlFor="title">Session Title *</Label>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Lecture 5: Data Structures"
                required
                className="mt-1.5"
              />
            </div>

            <div>
              <Label htmlFor="course">Course *</Label>
              <select
                id="course"
                value={courseId}
                onChange={(e) => setCourseId(e.target.value)}
                required
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm mt-1.5"
              >
                <option value="">Select a course...</option>
                {courses.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.title} ({c.term})
                  </option>
                ))}
              </select>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="scheduledAt">Date & Time *</Label>
                <Input
                  id="scheduledAt"
                  type="datetime-local"
                  value={scheduledAt}
                  onChange={(e) => setScheduledAt(e.target.value)}
                  required
                  className="mt-1.5"
                />
              </div>
              <div>
                <Label htmlFor="duration">Duration (minutes)</Label>
                <Input
                  id="duration"
                  type="number"
                  value={duration}
                  onChange={(e) => setDuration(e.target.value)}
                  min="15"
                  max="480"
                  className="mt-1.5"
                />
              </div>
            </div>

            {createSession.error && (
              <p className="text-sm text-red-600">
                {createSession.error instanceof Error
                  ? createSession.error.message
                  : 'Failed to schedule session'}
              </p>
            )}

            <div className="flex gap-3 pt-2">
              <Button type="submit" disabled={createSession.isPending}>
                {createSession.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {createSession.isPending ? 'Scheduling...' : 'Schedule Session'}
              </Button>
              <Button type="button" variant="outline" asChild>
                <Link href="/video">Cancel</Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
