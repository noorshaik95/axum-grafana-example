'use client'

import { useState } from 'react'
import Link from 'next/link'
import { ArrowLeft, Users } from 'lucide-react'

const studentBadges = [
  'Alex M.',
  'Jordan K.',
  'Sam T.',
  'Riley P.',
  'Casey W.',
  'Morgan L.',
  'Drew H.',
]

const rubricSummary = [
  { criterion: 'Correct base case handling', points: 2, maxPoints: 2 },
  { criterion: 'Recursive step is correct', points: 2, maxPoints: 3 },
  { criterion: 'Uses tail recursion', points: 0, maxPoints: 3 },
  { criterion: 'Code style & naming', points: 1, maxPoints: 1 },
  { criterion: 'Edge cases handled', points: 0, maxPoints: 1 },
]

export default function BatchGradePage() {
  const [feedback, setFeedback] = useState(
    `Your implementation produces correct output, but it's not tail recursive. When called on large lists, this will cause a stack overflow.\n\nTo fix this, use an accumulator parameter:\n\nsumList xs = go xs 0\n  where go [] acc = acc\n        go (y:ys) acc = go ys (acc + y)\n\nThis runs in O(1) stack space regardless of list size.`
  )
  const [applied, setApplied] = useState(false)

  const totalEarned = rubricSummary.reduce((s, r) => s + r.points, 0)
  const totalMax = rubricSummary.reduce((s, r) => s + r.maxPoints, 0)

  return (
    <div className="space-y-5 max-w-2xl">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link
          href="/grade/ps4"
          className="flex items-center gap-1.5 text-sm transition-colors hover:text-[#234e32]"
          style={{ color: '#6a6e62' }}
        >
          <ArrowLeft className="h-4 w-4" />
          Back
        </Link>
        <span style={{ color: '#e4e0d4' }}>·</span>
        <h1 className="font-serif text-2xl text-[#12170f]">Pattern C · 7 students</h1>
        <span
          className="rounded-full px-2.5 py-0.5 text-xs font-mono"
          style={{ background: 'rgba(255,182,72,0.2)', color: '#d97757' }}
        >
          stack overflow bug
        </span>
      </div>

      {/* Feedback template */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div
          className="flex items-center justify-between px-4 py-2.5 border-b"
          style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
        >
          <span className="text-xs font-mono font-medium" style={{ color: '#6a6e62' }}>
            FEEDBACK TEMPLATE
          </span>
          <span className="text-xs" style={{ color: '#6a6e62' }}>
            Editable — will be sent to all 7 students
          </span>
        </div>
        <textarea
          className="w-full px-5 py-4 text-sm leading-relaxed outline-none resize-none font-mono"
          style={{ background: '#fbfaf5', color: '#12170f', minHeight: 200 }}
          value={feedback}
          onChange={(e) => setFeedback(e.target.value)}
        />
      </div>

      {/* Rubric summary */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div
          className="flex items-center justify-between px-4 py-2.5 border-b"
          style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
        >
          <span className="text-xs font-mono font-medium" style={{ color: '#6a6e62' }}>
            RUBRIC SUMMARY
          </span>
          <span className="font-serif text-base font-bold" style={{ color: '#234e32' }}>
            {totalEarned}/{totalMax}
          </span>
        </div>
        <div className="divide-y" style={{ borderColor: '#e4e0d4' }}>
          {rubricSummary.map((item, i) => (
            <div key={i} className="flex items-center justify-between px-4 py-2.5">
              <span className="text-sm text-[#12170f]">{item.criterion}</span>
              <span
                className="font-mono text-xs rounded-full px-2 py-0.5"
                style={{
                  background: item.points === item.maxPoints ? '#dde9df' : 'rgba(255,182,72,0.15)',
                  color: item.points === item.maxPoints ? '#234e32' : '#d97757',
                }}
              >
                {item.points}/{item.maxPoints}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Student badges */}
      <div
        className="rounded-xl border px-5 py-4"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div className="flex items-center gap-2 mb-3">
          <Users className="h-4 w-4" style={{ color: '#6a6e62' }} />
          <span className="text-sm font-medium text-[#12170f]">Students in this batch</span>
        </div>
        <div className="flex flex-wrap gap-2">
          {studentBadges.map((name) => (
            <span
              key={name}
              className="rounded-full px-3 py-1 text-xs font-medium"
              style={{ background: '#f2f7f3', color: '#234e32', border: '1px solid #b8d2bd' }}
            >
              {name}
            </span>
          ))}
        </div>
      </div>

      {/* Apply button */}
      <button
        onClick={() => setApplied(true)}
        className="w-full rounded-xl py-3.5 text-sm font-semibold transition-all hover:opacity-80"
        style={
          applied
            ? { background: '#dde9df', color: '#234e32' }
            : { background: '#234e32', color: '#ffffff' }
        }
        disabled={applied}
      >
        {applied ? '✓ Applied to 7 students' : 'Apply to 7 students'}
      </button>
    </div>
  )
}
