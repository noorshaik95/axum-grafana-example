'use client'

import { useState } from 'react'
import { useSearchParams } from 'next/navigation'
import { ThumbsUp, Send, Loader2 } from 'lucide-react'
import { useLectureQuestions, useUpvoteQuestion } from '@/lib/api/hooks'

function formatAge(unixSec: number): string {
  const diffMs = Date.now() - unixSec * 1000
  if (diffMs < 60_000) return 'just now'
  const mins = Math.floor(diffMs / 60_000)
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  return `${hrs}h ago`
}

function shortName(userId: string): string {
  return userId.slice(0, 8)
}

export default function LectureLivePage() {
  const search = useSearchParams()
  const lectureId = search?.get('lectureId') ?? search?.get('id') ?? ''

  const questionsQuery = useLectureQuestions(lectureId)
  const upvote = useUpvoteQuestion()

  const [newQuestion, setNewQuestion] = useState('')

  const questions = questionsQuery.data?.questions ?? []
  const sorted = [...questions].sort((a, b) => b.upvotes - a.upvotes)

  const handleUpvote = (questionId: string) => {
    if (!lectureId) return
    upvote.mutate({ lectureId, questionId })
  }

  return (
    <div
      className="grid grid-cols-1 gap-5 lg:grid-cols-3"
      style={{ minHeight: 'calc(100vh - 140px)' }}
    >
      {/* Left: Video area — 2/3 */}
      <div className="lg:col-span-2 flex flex-col gap-4">
        <div
          className="relative rounded-2xl overflow-hidden flex items-center justify-center"
          style={{ background: '#0f2617', aspectRatio: '16/9' }}
        >
          <div className="absolute top-4 left-4 flex items-center gap-2">
            <span
              className="flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-mono font-medium"
              style={{ background: 'rgba(217,119,87,0.9)', color: '#ffffff' }}
            >
              <span className="h-1.5 w-1.5 rounded-full bg-current animate-pulse" />
              LIVE
            </span>
          </div>
          <div className="text-center">
            <div className="text-6xl mb-3" style={{ opacity: 0.15 }}>
              🎥
            </div>
            <p className="font-serif text-xl" style={{ color: 'rgba(255,255,255,0.4)' }}>
              Camera feed
            </p>
          </div>
          <div className="absolute bottom-4 left-1/2 -translate-x-1/2 flex items-center gap-2">
            {['Mute', 'Video off', 'Share screen', 'End'].map((ctrl) => (
              <button
                key={ctrl}
                className="rounded-xl px-3 py-2 text-xs font-medium transition-all hover:opacity-80"
                style={
                  ctrl === 'End'
                    ? { background: '#d97757', color: '#ffffff' }
                    : { background: 'rgba(255,255,255,0.15)', color: 'rgba(255,255,255,0.85)' }
                }
              >
                {ctrl}
              </button>
            ))}
          </div>
        </div>

        <div
          className="rounded-xl border px-5 py-4"
          style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
        >
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-mono font-medium" style={{ color: '#6a6e62' }}>
              LECTURE
            </span>
            <span className="text-xs font-mono" style={{ color: '#6a6e62' }}>
              {lectureId ? lectureId : 'no lecture id'}
            </span>
          </div>
          <p className="text-sm mt-1" style={{ color: '#6a6e62' }}>
            {lectureId
              ? 'Live session in progress.'
              : 'Pass ?lectureId=... to open a live lecture feed.'}
          </p>
        </div>
      </div>

      {/* Right: Questions list — 1/3 */}
      <div
        className="flex flex-col rounded-xl border overflow-hidden"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div
          className="flex items-center justify-between px-4 py-3 border-b"
          style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
        >
          <span className="text-sm font-semibold text-[#12170f]">Questions</span>
          <div className="flex items-center gap-1.5">
            <span
              className="rounded-full px-2 py-0.5 text-xs font-mono"
              style={{ background: '#dde9df', color: '#234e32' }}
            >
              {sorted.length} total
            </span>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto divide-y" style={{ borderColor: '#e4e0d4' }}>
          {!lectureId ? (
            <div className="px-4 py-6 text-sm text-center" style={{ color: '#6a6e62' }}>
              No lecture selected.
            </div>
          ) : questionsQuery.isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-indigo-600" />
            </div>
          ) : questionsQuery.error ? (
            <div className="px-4 py-6 text-sm text-center text-red-700">
              Failed to load questions.
            </div>
          ) : sorted.length === 0 ? (
            <div className="px-4 py-6 text-sm text-center" style={{ color: '#6a6e62' }}>
              No questions yet.
            </div>
          ) : (
            sorted.map((q) => (
              <div key={q.id} className="px-4 py-3">
                <div className="flex items-start justify-between gap-2 mb-1.5">
                  <div className="flex items-center gap-1.5">
                    <span className="text-xs font-medium text-[#12170f]">
                      {shortName(q.user_id)}
                    </span>
                    <span className="text-xs" style={{ color: '#6a6e62' }}>
                      {formatAge(q.submitted_at_unix)}
                    </span>
                  </div>
                </div>
                <p className="text-sm text-[#12170f] leading-snug mb-2">{q.text}</p>
                <button
                  onClick={() => handleUpvote(q.id)}
                  disabled={upvote.isPending}
                  className="flex items-center gap-1 text-xs transition-colors hover:text-[#234e32] disabled:opacity-50"
                  style={{ color: '#6a6e62' }}
                >
                  <ThumbsUp className="h-3 w-3" />
                  {q.upvotes}
                </button>
              </div>
            ))
          )}
        </div>

        <div className="border-t p-3" style={{ borderColor: '#e4e0d4' }}>
          <div
            className="flex items-center gap-2 rounded-lg border px-3 py-2"
            style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
          >
            <input
              type="text"
              placeholder="Pin a question..."
              className="flex-1 text-sm bg-transparent outline-none"
              style={{ color: '#12170f' }}
              value={newQuestion}
              onChange={(e) => setNewQuestion(e.target.value)}
            />
            <button>
              <Send className="h-4 w-4" style={{ color: '#6a6e62' }} />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
