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
import { useCreateAnnouncement, useInstructorCourses } from '@/lib/api/hooks'
import { useProfile } from '../../../../../shared/lib/api/hooks'
import type { Course } from '../../../../../shared/lib/api/types'

export default function CreateAnnouncementPage() {
  const router = useRouter()
  const { data: profile } = useProfile()
  const { data: coursesData } = useInstructorCourses(
    profile ? { instructorId: profile.id } : undefined
  )
  const createAnnouncement = useCreateAnnouncement()

  const courses: readonly Course[] = coursesData?.data ?? []

  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [courseId, setCourseId] = useState('')
  const [scheduledAt, setScheduledAt] = useState('')
  const [publishNow, setPublishNow] = useState(true)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim() || !content.trim()) return
    try {
      await createAnnouncement.mutateAsync({
        title: title.trim(),
        content: content.trim(),
        courseId: courseId || undefined,
        scheduledAt: !publishNow && scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
        publish: publishNow,
      })
      router.push('/announcements')
    } catch {
      // error via mutation
    }
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
          <Link href="/announcements">
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back to Announcements
          </Link>
        </Button>
        <h1 className="text-2xl font-bold text-slate-900">New Announcement</h1>
        <p className="text-slate-500 mt-1">Create an announcement for your students.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Announcement Details</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <Label htmlFor="title">Title *</Label>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Important Update"
                required
                className="mt-1.5"
              />
            </div>

            <div>
              <Label htmlFor="content">Content *</Label>
              <Textarea
                id="content"
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder="Write your announcement here..."
                rows={8}
                required
                className="mt-1.5"
              />
            </div>

            <div>
              <Label htmlFor="course">Course (optional)</Label>
              <select
                id="course"
                value={courseId}
                onChange={(e) => setCourseId(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm mt-1.5"
              >
                <option value="">All courses</option>
                {courses.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.title}
                  </option>
                ))}
              </select>
            </div>

            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="publishNow"
                  checked={publishNow}
                  onChange={(e) => setPublishNow(e.target.checked)}
                  className="rounded border-slate-300"
                />
                <Label htmlFor="publishNow">Publish immediately</Label>
              </div>

              {!publishNow && (
                <div>
                  <Label htmlFor="scheduledAt">Schedule for</Label>
                  <Input
                    id="scheduledAt"
                    type="datetime-local"
                    value={scheduledAt}
                    onChange={(e) => setScheduledAt(e.target.value)}
                    className="mt-1.5"
                  />
                </div>
              )}
            </div>

            {createAnnouncement.error && (
              <p className="text-sm text-red-600">
                {createAnnouncement.error instanceof Error
                  ? createAnnouncement.error.message
                  : 'Failed to create announcement'}
              </p>
            )}

            <div className="flex gap-3 pt-2">
              <Button type="submit" disabled={createAnnouncement.isPending}>
                {createAnnouncement.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {publishNow ? 'Publish' : 'Schedule'}
              </Button>
              <Button type="button" variant="outline" asChild>
                <Link href="/announcements">Cancel</Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
