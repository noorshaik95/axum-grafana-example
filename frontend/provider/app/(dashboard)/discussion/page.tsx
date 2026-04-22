'use client'

import { useMemo, useState } from 'react'
import { MessageSquare, Clock, Loader2, Plus } from 'lucide-react'
import {
  useInstructorCourses,
  useDiscussionThreads,
  useCreateDiscussionThread,
} from '@/lib/api/hooks'
import type { DiscussionThread } from '@/lib/api/discussion'

type Segment = 'needs-reply' | 'all' | 'unanswered'

function waitColor(hours: number): { bg: string; color: string } {
  if (hours >= 12) return { bg: 'rgba(217,119,87,0.15)', color: '#d97757' }
  if (hours >= 6) return { bg: 'rgba(255,182,72,0.15)', color: '#c48d1a' }
  return { bg: '#f2f7f3', color: '#234e32' }
}

function waitLabel(hours: number): string {
  if (hours < 1) return '<1h'
  if (hours < 24) return `${Math.round(hours)}h`
  return `${Math.round(hours / 24)}d`
}

function hoursSince(iso: string): number {
  try {
    return (Date.now() - new Date(iso).getTime()) / 3_600_000
  } catch {
    return 0
  }
}

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase())
    .join('')
}

export default function DiscussionPage() {
  const coursesQuery = useInstructorCourses({ pageSize: 50 })
  const courses = coursesQuery.data?.data ?? []
  const [courseId, setCourseId] = useState<string>('')
  const effectiveCourseId = courseId || courses[0]?.id || ''

  const threadsQuery = useDiscussionThreads(effectiveCourseId || undefined)
  const createThread = useCreateDiscussionThread()

  const [segment, setSegment] = useState<Segment>('all')
  const [composing, setComposing] = useState(false)
  const [draft, setDraft] = useState({ title: '', body: '' })

  const threads = threadsQuery.data?.threads ?? []

  const displayThreads = useMemo(() => {
    if (segment === 'needs-reply') return threads.filter((t) => t.reply_count === 0)
    if (segment === 'unanswered')
      return threads.filter((t) => t.reply_count === 0 && hoursSince(t.last_activity_at) >= 8)
    return threads
  }, [threads, segment])

  const segmentTabs = [
    {
      id: 'needs-reply' as Segment,
      label: 'Needs reply',
      count: threads.filter((t) => t.reply_count === 0).length,
    },
    { id: 'all' as Segment, label: 'All', count: threads.length },
    {
      id: 'unanswered' as Segment,
      label: 'Unanswered',
      count: threads.filter((t) => t.reply_count === 0 && hoursSince(t.last_activity_at) >= 8)
        .length,
    },
  ]

  const handleCreate = async () => {
    if (!effectiveCourseId || !draft.title.trim() || !draft.body.trim()) return
    await createThread.mutateAsync({
      course_id: effectiveCourseId,
      title: draft.title,
      initial_post: draft.body,
    })
    setDraft({ title: '', body: '' })
    setComposing(false)
  }

  if (coursesQuery.isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
      </div>
    )
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h1 className="font-serif text-2xl text-[#12170f]">Discussion</h1>
          <p className="text-sm mt-1" style={{ color: '#6a6e62' }}>
            Student threads — sorted by urgency
          </p>
        </div>
        <div className="flex items-center gap-2">
          {courses.length > 0 && (
            <select
              value={effectiveCourseId}
              onChange={(e) => setCourseId(e.target.value)}
              className="rounded-lg border px-3 py-2 text-sm"
              style={{ background: '#ffffff', borderColor: '#e4e0d4', color: '#12170f' }}
            >
              {courses.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.title}
                </option>
              ))}
            </select>
          )}
          <button
            onClick={() => setComposing((v) => !v)}
            disabled={!effectiveCourseId}
            className="flex items-center gap-1.5 rounded-xl px-3 py-2 text-sm font-semibold transition-all hover:opacity-80 disabled:opacity-50"
            style={{ background: '#234e32', color: '#ffffff' }}
          >
            <Plus className="h-4 w-4" />
            New thread
          </button>
        </div>
      </div>

      {composing && (
        <div
          className="rounded-xl border p-4 space-y-3"
          style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
        >
          <input
            type="text"
            placeholder="Thread title"
            value={draft.title}
            onChange={(e) => setDraft((d) => ({ ...d, title: e.target.value }))}
            className="w-full rounded-lg border px-3 py-2 text-sm outline-none"
            style={{ borderColor: '#e4e0d4', color: '#12170f' }}
          />
          <textarea
            placeholder="Initial post..."
            value={draft.body}
            onChange={(e) => setDraft((d) => ({ ...d, body: e.target.value }))}
            className="w-full rounded-lg border px-3 py-2 text-sm outline-none resize-none"
            style={{ borderColor: '#e4e0d4', color: '#12170f', minHeight: 100 }}
          />
          <div className="flex justify-end gap-2">
            <button
              onClick={() => {
                setComposing(false)
                setDraft({ title: '', body: '' })
              }}
              className="rounded-xl border px-4 py-2 text-xs font-medium"
              style={{ borderColor: '#e4e0d4', color: '#6a6e62' }}
            >
              Cancel
            </button>
            <button
              onClick={handleCreate}
              disabled={createThread.isPending || !draft.title.trim() || !draft.body.trim()}
              className="rounded-xl px-4 py-2 text-xs font-semibold transition-all hover:opacity-80 disabled:opacity-50"
              style={{ background: '#234e32', color: '#ffffff' }}
            >
              {createThread.isPending ? 'Posting...' : 'Post thread'}
            </button>
          </div>
        </div>
      )}

      {/* Segment tabs */}
      <div
        className="inline-flex items-center rounded-xl p-1 gap-1"
        style={{ background: '#f6f3ec' }}
      >
        {segmentTabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setSegment(tab.id)}
            className="flex items-center gap-1.5 rounded-lg px-4 py-1.5 text-sm font-medium transition-all"
            style={
              segment === tab.id
                ? { background: '#234e32', color: '#ffffff' }
                : { color: '#6a6e62' }
            }
          >
            {tab.label}
            <span
              className="rounded-full px-1.5 py-0.5 text-xs font-mono min-w-[20px] text-center"
              style={
                segment === tab.id
                  ? { background: 'rgba(255,255,255,0.2)', color: '#ffffff' }
                  : { background: '#e4e0d4', color: '#6a6e62' }
              }
            >
              {tab.count}
            </span>
          </button>
        ))}
      </div>

      {threadsQuery.isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
        </div>
      ) : threadsQuery.error ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          Failed to load threads.
        </div>
      ) : displayThreads.length === 0 ? (
        <div
          className="rounded-xl border px-5 py-10 text-center"
          style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
        >
          <MessageSquare className="h-8 w-8 mx-auto mb-2" style={{ color: '#e4e0d4' }} />
          <p className="text-sm" style={{ color: '#6a6e62' }}>
            No threads here.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {displayThreads.map((thread: DiscussionThread) => {
            const waitHours = hoursSince(thread.last_activity_at)
            const isReplied = thread.reply_count > 0
            const wc = waitColor(waitHours)
            return (
              <div
                key={thread.id}
                className="rounded-xl border overflow-hidden"
                style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
              >
                <div className="flex items-start gap-4 px-5 py-4">
                  <div
                    className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-semibold mt-0.5"
                    style={{
                      background: isReplied ? '#f2f7f3' : '#dde9df',
                      color: isReplied ? '#6a6e62' : '#234e32',
                    }}
                  >
                    {initials(thread.created_by) || '?'}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-[#12170f] mb-0.5">{thread.title}</p>
                    <p className="text-xs" style={{ color: '#6a6e62' }}>
                      {thread.reply_count} {thread.reply_count === 1 ? 'reply' : 'replies'} · last
                      activity {waitLabel(waitHours)} ago
                    </p>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    {!isReplied && (
                      <span
                        className="flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-mono"
                        style={{ background: wc.bg, color: wc.color }}
                      >
                        <Clock className="h-3 w-3" />
                        waiting {waitLabel(waitHours)}
                      </span>
                    )}
                    {isReplied && (
                      <span
                        className="rounded-full px-2.5 py-1 text-xs font-mono"
                        style={{ background: '#dde9df', color: '#234e32' }}
                      >
                        {thread.reply_count} posts
                      </span>
                    )}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
