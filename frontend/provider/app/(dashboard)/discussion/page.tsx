'use client'

import { useState } from 'react'
import { MessageSquare, Clock } from 'lucide-react'

type Segment = 'needs-reply' | 'all' | 'unanswered'

const threads = [
  {
    id: 1,
    student: 'Jordan Kim',
    initials: 'JK',
    subject: 'PS4 Q3 — confused about accumulator',
    preview: 'I tried adding an acc parameter but my function still doesnt type-check...',
    waitHours: 14,
    replied: false,
    course: 'CS 3110',
  },
  {
    id: 2,
    student: 'Sam Torres',
    initials: 'ST',
    subject: 'Question about lecture 6 slides',
    preview: 'The slide about lazy evaluation confused me — can you clarify what thunks are?',
    waitHours: 9,
    replied: false,
    course: 'CS 3110',
  },
  {
    id: 3,
    student: 'Riley Park',
    initials: 'RP',
    subject: 'Grade correction request for HW2',
    preview: 'I believe my answer for Q4 was graded incorrectly — I showed the derivation...',
    waitHours: 6,
    replied: false,
    course: 'CS 3110',
  },
  {
    id: 4,
    student: 'Casey Walsh',
    initials: 'CW',
    subject: 'Office hours — can I come today?',
    preview: 'I know your hours are 2-3pm but I have a conflict. Any chance you have 5 min at...',
    waitHours: 3,
    replied: false,
    course: 'CS 3110',
  },
  {
    id: 5,
    student: 'Morgan Lee',
    initials: 'ML',
    subject: 'Thank you for the PS3 feedback',
    preview: 'Really appreciated the detailed comments. The suggestion about type inference...',
    waitHours: 1,
    replied: false,
    course: 'CS 3110',
  },
  {
    id: 6,
    student: 'Drew Harris',
    initials: 'DH',
    subject: 'Extra credit opportunity?',
    preview: 'Someone mentioned there might be extra credit for the final project...',
    waitHours: 0,
    replied: true,
    course: 'CS 3110',
  },
]

const segmentTabs = [
  {
    id: 'needs-reply' as Segment,
    label: 'Needs reply',
    count: threads.filter((t) => !t.replied).length,
  },
  { id: 'all' as Segment, label: 'All', count: threads.length },
  {
    id: 'unanswered' as Segment,
    label: 'Unanswered',
    count: threads.filter((t) => !t.replied && t.waitHours > 8).length,
  },
]

function waitColor(hours: number): { bg: string; color: string } {
  if (hours >= 12) return { bg: 'rgba(217,119,87,0.15)', color: '#d97757' }
  if (hours >= 6) return { bg: 'rgba(255,182,72,0.15)', color: '#c48d1a' }
  return { bg: '#f2f7f3', color: '#234e32' }
}

function waitLabel(hours: number): string {
  if (hours === 0) return 'just now'
  if (hours < 1) return '<1h'
  return `${hours}h`
}

export default function DiscussionPage() {
  const [segment, setSegment] = useState<Segment>('needs-reply')
  const [replying, setReplying] = useState<number | null>(null)
  const [replied, setReplied] = useState<Set<number>>(new Set())

  const displayThreads = threads.filter((t) => {
    if (segment === 'needs-reply') return !t.replied && !replied.has(t.id)
    if (segment === 'unanswered') return !t.replied && !replied.has(t.id) && t.waitHours >= 8
    return true
  })

  const handleReply = (id: number) => {
    setReplied((prev) => new Set([...prev, id]))
    setReplying(null)
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div>
        <h1 className="font-serif text-2xl text-[#12170f]">Discussion</h1>
        <p className="text-sm mt-1" style={{ color: '#6a6e62' }}>
          Student threads — sorted by urgency
        </p>
      </div>

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

      {/* Thread list */}
      <div className="space-y-2">
        {displayThreads.length === 0 ? (
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
          displayThreads.map((thread) => {
            const isReplied = thread.replied || replied.has(thread.id)
            const wc = waitColor(thread.waitHours)
            const isOpen = replying === thread.id
            return (
              <div
                key={thread.id}
                className="rounded-xl border overflow-hidden transition-all"
                style={{
                  background: '#ffffff',
                  borderColor: isOpen ? '#8ab694' : '#e4e0d4',
                }}
              >
                <div
                  className="flex items-start gap-4 px-5 py-4 cursor-pointer"
                  onClick={() => setReplying(isOpen ? null : thread.id)}
                >
                  {/* Avatar */}
                  <div
                    className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-semibold mt-0.5"
                    style={{
                      background: isReplied ? '#f2f7f3' : '#dde9df',
                      color: isReplied ? '#6a6e62' : '#234e32',
                    }}
                  >
                    {thread.initials}
                  </div>

                  {/* Content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-0.5">
                      <span
                        className="text-sm font-semibold"
                        style={{ color: isReplied ? '#6a6e62' : '#12170f' }}
                      >
                        {thread.student}
                      </span>
                      <span className="text-xs" style={{ color: '#6a6e62' }}>
                        · {thread.course}
                      </span>
                    </div>
                    <p
                      className="text-sm font-medium mb-0.5"
                      style={{ color: isReplied ? '#6a6e62' : '#12170f' }}
                    >
                      {thread.subject}
                    </p>
                    <p className="text-xs truncate" style={{ color: '#6a6e62' }}>
                      {thread.preview}
                    </p>
                  </div>

                  {/* Wait time + reply */}
                  <div className="flex items-center gap-2 shrink-0">
                    {!isReplied && (
                      <span
                        className="flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-mono"
                        style={{ background: wc.bg, color: wc.color }}
                      >
                        <Clock className="h-3 w-3" />
                        waiting {waitLabel(thread.waitHours)}
                      </span>
                    )}
                    {isReplied ? (
                      <span
                        className="rounded-full px-2.5 py-1 text-xs font-mono"
                        style={{ background: '#dde9df', color: '#234e32' }}
                      >
                        ✓ replied
                      </span>
                    ) : (
                      <button
                        className="rounded-xl px-3 py-1.5 text-xs font-semibold transition-all hover:opacity-80"
                        style={
                          thread.waitHours >= 12
                            ? { background: '#d97757', color: '#ffffff' }
                            : thread.waitHours >= 6
                              ? { background: '#ffb648', color: '#12170f' }
                              : { background: '#234e32', color: '#ffffff' }
                        }
                        onClick={(e) => {
                          e.stopPropagation()
                          setReplying(isOpen ? null : thread.id)
                        }}
                      >
                        Reply
                      </button>
                    )}
                  </div>
                </div>

                {/* Expanded reply area */}
                {isOpen && !isReplied && (
                  <div
                    className="border-t px-5 py-4"
                    style={{ borderColor: '#e4e0d4', background: '#fbfaf5' }}
                  >
                    <textarea
                      className="w-full rounded-xl border px-4 py-3 text-sm outline-none resize-none"
                      style={{
                        background: '#ffffff',
                        borderColor: '#e4e0d4',
                        color: '#12170f',
                        minHeight: 80,
                      }}
                      placeholder={`Reply to ${thread.student}...`}
                      autoFocus
                    />
                    <div className="flex justify-end gap-2 mt-3">
                      <button
                        onClick={() => setReplying(null)}
                        className="rounded-xl border px-4 py-2 text-xs font-medium transition-colors hover:bg-[#f6f3ec]"
                        style={{ borderColor: '#e4e0d4', color: '#6a6e62' }}
                      >
                        Cancel
                      </button>
                      <button
                        onClick={() => handleReply(thread.id)}
                        className="rounded-xl px-4 py-2 text-xs font-semibold transition-all hover:opacity-80"
                        style={{ background: '#234e32', color: '#ffffff' }}
                      >
                        Send reply
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
