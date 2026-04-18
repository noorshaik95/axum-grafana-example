'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, ChevronLeft, ChevronRight, MessageSquare } from 'lucide-react'

const rubricItems = [
  { id: 1, criterion: 'Correct base case handling', points: 2, maxPoints: 2, checked: true },
  { id: 2, criterion: 'Recursive step is correct', points: 3, maxPoints: 3, checked: true },
  {
    id: 3,
    criterion: 'Uses tail recursion',
    points: 3,
    maxPoints: 3,
    checked: false,
    failed: true,
  },
  { id: 4, criterion: 'Code style & naming', points: 1, maxPoints: 1, checked: true },
  {
    id: 5,
    criterion: 'Edge cases handled',
    points: 1,
    maxPoints: 1,
    checked: false,
    failed: false,
  },
]

const mockCode = `-- Problem Set 4, Question 3
-- Tail-recursive sum

-- Student implementation:
sumList :: [Int] -> Int
sumList [] = 0
sumList (x:xs) = x + sumList xs

-- Note: This is naive recursion, not tail recursive.
-- A tail-recursive version would use an accumulator:
-- sumList xs = go xs 0
--   where go [] acc = acc
--         go (y:ys) acc = go ys (acc + y)`

export default function SubmissionGradePage() {
  const params = useParams()
  const assignmentId = params.assignmentId as string
  const submissionId = params.submissionId as string
  const [checkedItems, setCheckedItems] = useState<Set<number>>(
    new Set(rubricItems.filter((r) => r.checked).map((r) => r.id))
  )
  const [comment, setComment] = useState('')
  const [showComment, setShowComment] = useState(false)

  const totalEarned = rubricItems.reduce(
    (sum, item) => sum + (checkedItems.has(item.id) ? item.points : 0),
    0
  )
  const totalMax = rubricItems.reduce((sum, item) => sum + item.maxPoints, 0)

  const toggleItem = (id: number) => {
    setCheckedItems((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link
            href={`/grade/${assignmentId}`}
            className="flex items-center gap-1.5 text-sm transition-colors hover:text-[#234e32]"
            style={{ color: '#6a6e62' }}
          >
            <ArrowLeft className="h-4 w-4" />
            Back to queue
          </Link>
          <span style={{ color: '#e4e0d4' }}>·</span>
          <h1 className="font-serif text-xl text-[#12170f]">17 of 47 · Jon L.</h1>
          <span
            className="rounded-full px-2.5 py-0.5 text-xs font-mono"
            style={{ background: '#dde9df', color: '#234e32' }}
          >
            naive recursion
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs font-mono" style={{ color: '#6a6e62' }}>
            j/k prev/next
          </span>
          <div className="flex items-center gap-1">
            <Link
              href={`/grade/${assignmentId}/prev`}
              className="flex h-8 w-8 items-center justify-center rounded-lg border transition-colors hover:bg-[#f6f3ec]"
              style={{ borderColor: '#e4e0d4' }}
            >
              <ChevronLeft className="h-4 w-4" style={{ color: '#6a6e62' }} />
            </Link>
            <Link
              href={`/grade/${assignmentId}/next`}
              className="flex h-8 w-8 items-center justify-center rounded-lg border transition-colors hover:bg-[#f6f3ec]"
              style={{ borderColor: '#e4e0d4' }}
            >
              <ChevronRight className="h-4 w-4" style={{ color: '#6a6e62' }} />
            </Link>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        {/* Code block — 3 cols */}
        <div className="lg:col-span-3">
          <div className="rounded-xl overflow-hidden border" style={{ borderColor: '#1a3a26' }}>
            {/* Code header */}
            <div
              className="flex items-center justify-between px-4 py-2.5 border-b"
              style={{ background: '#0f2617', borderColor: '#1a3a26' }}
            >
              <span className="font-mono text-xs" style={{ color: 'rgba(255,255,255,0.5)' }}>
                submission_{submissionId}.hs
              </span>
              <span className="font-mono text-xs" style={{ color: 'rgba(255,255,255,0.35)' }}>
                Haskell
              </span>
            </div>
            {/* Code content */}
            <pre
              className="p-5 text-sm font-mono leading-relaxed overflow-x-auto"
              style={{ background: '#12170f', color: 'rgba(255,255,255,0.85)', minHeight: 320 }}
            >
              <code>
                {mockCode.split('\n').map((line, i) => (
                  <div key={i} className="flex">
                    <span
                      className="select-none text-right pr-4 shrink-0 w-8"
                      style={{ color: 'rgba(255,255,255,0.2)', fontSize: '11px' }}
                    >
                      {i + 1}
                    </span>
                    <span>
                      {line.startsWith('--') ? (
                        <span style={{ color: '#5d9a6c' }}>{line}</span>
                      ) : line.includes('::') ? (
                        <span>
                          {line.split('::').map((part, j) => (
                            <span key={j}>
                              {j === 0 ? (
                                <span style={{ color: '#8ab694' }}>{part}</span>
                              ) : (
                                <span>
                                  <span style={{ color: 'rgba(255,255,255,0.4)' }}>::</span>
                                  <span style={{ color: '#ffb648' }}>{part}</span>
                                </span>
                              )}
                            </span>
                          ))}
                        </span>
                      ) : (
                        line
                      )}
                    </span>
                  </div>
                ))}
              </code>
            </pre>
          </div>
        </div>

        {/* Rubric panel — 2 cols */}
        <div className="lg:col-span-2 flex flex-col gap-4">
          {/* Score summary */}
          <div
            className="rounded-xl border px-5 py-4 flex items-center justify-between"
            style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
          >
            <span className="text-sm font-medium text-[#12170f]">Score</span>
            <span className="font-serif text-2xl font-bold text-[#234e32]">
              {totalEarned}
              <span className="text-base font-normal text-[#6a6e62]">/{totalMax}</span>
            </span>
          </div>

          {/* Rubric rows */}
          <div
            className="rounded-xl border overflow-hidden"
            style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
          >
            <div
              className="px-4 py-2.5 border-b text-xs font-mono font-medium"
              style={{ background: '#f6f3ec', borderColor: '#e4e0d4', color: '#6a6e62' }}
            >
              RUBRIC
            </div>
            <div className="divide-y" style={{ borderColor: '#e4e0d4' }}>
              {rubricItems.map((item) => {
                const isChecked = checkedItems.has(item.id)
                const isFailed = item.failed && !isChecked
                return (
                  <div
                    key={item.id}
                    className="flex items-center gap-3 px-4 py-3 cursor-pointer transition-colors"
                    style={{
                      background: isFailed
                        ? 'rgba(255,182,72,0.08)'
                        : isChecked
                          ? 'rgba(35,78,50,0.04)'
                          : 'transparent',
                    }}
                    onClick={() => toggleItem(item.id)}
                  >
                    {/* Checkbox */}
                    <div
                      className="flex h-5 w-5 shrink-0 items-center justify-center rounded border-2 transition-all"
                      style={{
                        background: isChecked ? '#2d6640' : 'transparent',
                        borderColor: isChecked ? '#2d6640' : '#e4e0d4',
                      }}
                    >
                      {isChecked && (
                        <svg
                          className="h-3 w-3 text-white"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                          strokeWidth={3}
                        >
                          <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                      )}
                    </div>
                    <span
                      className="flex-1 text-sm"
                      style={{ color: isChecked ? '#12170f' : '#6a6e62' }}
                    >
                      {item.criterion}
                    </span>
                    <span
                      className="font-mono text-xs rounded-full px-2 py-0.5 shrink-0"
                      style={{
                        background: isChecked ? '#dde9df' : '#f6f3ec',
                        color: isChecked ? '#234e32' : '#6a6e62',
                      }}
                    >
                      {item.points}/{item.maxPoints}
                    </span>
                  </div>
                )
              })}
            </div>
          </div>

          {/* Comment area */}
          {showComment && (
            <div
              className="rounded-xl border overflow-hidden"
              style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
            >
              <textarea
                className="w-full px-4 py-3 text-sm resize-none outline-none"
                style={{ background: 'transparent', color: '#12170f', minHeight: 100 }}
                placeholder="Add feedback for this student..."
                value={comment}
                onChange={(e) => setComment(e.target.value)}
              />
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-2">
            <Link
              href={`/grade/${assignmentId}/next`}
              className="flex-1 flex items-center justify-center gap-2 rounded-xl px-4 py-3 text-sm font-semibold transition-all hover:opacity-80"
              style={{ background: '#234e32', color: '#ffffff' }}
            >
              Save & next ↵
            </Link>
            <button
              onClick={() => setShowComment(!showComment)}
              className="flex items-center gap-2 rounded-xl border px-4 py-3 text-sm font-medium transition-colors hover:bg-[#f6f3ec]"
              style={{ borderColor: '#e4e0d4', color: '#12170f' }}
            >
              <MessageSquare className="h-4 w-4" />
              Comment
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
