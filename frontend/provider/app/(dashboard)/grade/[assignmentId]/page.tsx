'use client'

import Link from 'next/link'
import { useParams } from 'next/navigation'
import { ArrowLeft, CheckCircle2, AlertCircle, Zap } from 'lucide-react'

const patterns = [
  {
    id: 'A',
    label: 'tail recursion ✓',
    count: 23,
    description: 'Auto-tests pass — correct implementation with proper tail call optimization.',
    action: '1-click full credit',
    actionLabel: 'Full credit all',
    bg: '#f2f7f3',
    border: '#b8d2bd',
    badgeBg: '#234e32',
    badgeText: '#ffffff',
    icon: CheckCircle2,
    iconColor: '#3e7d4f',
    quick: true,
  },
  {
    id: 'B',
    label: 'naive recursion',
    count: 14,
    description: 'Correct output but not tail-recursive. Rubric deduction applies.',
    action: '8/10 rubric',
    actionLabel: 'Batch grade',
    bg: '#ffffff',
    border: '#e4e0d4',
    badgeBg: '#dde9df',
    badgeText: '#234e32',
    icon: CheckCircle2,
    iconColor: '#5d9a6c',
    quick: false,
  },
  {
    id: 'C',
    label: 'stack overflow bug',
    count: 7,
    description: 'Same overflow on large inputs. Shared batch feedback opportunity.',
    action: 'shared bug',
    actionLabel: 'Batch feedback',
    bg: 'rgba(255,182,72,0.06)',
    border: 'rgba(255,182,72,0.4)',
    badgeBg: '#ffb648',
    badgeText: '#12170f',
    icon: AlertCircle,
    iconColor: '#d97757',
    quick: false,
  },
  {
    id: 'D',
    label: 'one-off',
    count: 3,
    description: 'Unique approaches — needs individual review.',
    action: 'individual',
    actionLabel: 'Review each',
    bg: 'rgba(217,119,87,0.06)',
    border: 'rgba(217,119,87,0.35)',
    badgeBg: '#d97757',
    badgeText: '#ffffff',
    icon: AlertCircle,
    iconColor: '#d97757',
    quick: false,
  },
]

const segments = ['By pattern', 'Alphabetical', 'Submitted']

export default function GradeAssignmentPage() {
  const params = useParams()
  const assignmentId = params.assignmentId as string
  const displayId = assignmentId?.toUpperCase() ?? 'PS4'

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link
          href="/teach"
          className="flex items-center gap-1.5 text-sm transition-colors hover:text-[#234e32]"
          style={{ color: '#6a6e62' }}
        >
          <ArrowLeft className="h-4 w-4" />
          Back
        </Link>
        <span style={{ color: '#e4e0d4' }}>·</span>
        <h1 className="font-serif text-2xl text-[#12170f]">{displayId} · 47 to grade</h1>
      </div>

      {/* Segment control */}
      <div
        className="inline-flex items-center rounded-xl p-1 gap-1"
        style={{ background: '#f6f3ec' }}
      >
        {segments.map((seg, i) => (
          <button
            key={seg}
            className="rounded-lg px-4 py-1.5 text-sm font-medium transition-all"
            style={i === 0 ? { background: '#234e32', color: '#ffffff' } : { color: '#6a6e62' }}
          >
            {seg}
          </button>
        ))}
      </div>

      {/* Pattern cards */}
      <div className="space-y-3">
        {patterns.map((pattern) => {
          const Icon = pattern.icon
          return (
            <div
              key={pattern.id}
              className="flex items-center gap-4 rounded-xl border px-5 py-4"
              style={{ background: pattern.bg, borderColor: pattern.border }}
            >
              <div
                className="flex h-10 w-10 items-center justify-center rounded-lg shrink-0"
                style={{ background: pattern.badgeBg + '22' }}
              >
                <Icon className="h-5 w-5" style={{ color: pattern.iconColor }} />
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-semibold text-sm text-[#12170f]">Pattern {pattern.id}</span>
                  <span
                    className="rounded-full px-2.5 py-0.5 text-xs font-mono font-medium"
                    style={{ background: pattern.badgeBg, color: pattern.badgeText }}
                  >
                    {pattern.label} ({pattern.count})
                  </span>
                  {pattern.quick && (
                    <span
                      className="flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-mono"
                      style={{ background: '#dde9df', color: '#234e32' }}
                    >
                      <Zap className="h-3 w-3" />
                      quick
                    </span>
                  )}
                </div>
                <p className="text-sm" style={{ color: '#6a6e62' }}>
                  {pattern.description}
                </p>
              </div>

              <Link
                href={`/grade/${assignmentId}/batch?pattern=${pattern.id}`}
                className="shrink-0 rounded-xl px-4 py-2 text-sm font-semibold transition-all hover:opacity-80"
                style={
                  pattern.quick
                    ? { background: '#234e32', color: '#ffffff' }
                    : { background: '#f6f3ec', color: '#12170f', border: '1px solid #e4e0d4' }
                }
              >
                {pattern.actionLabel}
              </Link>
            </div>
          )
        })}
      </div>

      {/* Summary bar */}
      <div
        className="rounded-xl border px-5 py-3 flex items-center justify-between"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div className="flex items-center gap-4 text-sm">
          <span style={{ color: '#6a6e62' }}>
            <span className="font-semibold text-[#12170f]">47</span> total
          </span>
          <span style={{ color: '#6a6e62' }}>
            <span className="font-semibold text-[#234e32]">0</span> graded
          </span>
          <span style={{ color: '#6a6e62' }}>
            <span className="font-semibold text-[#d97757]">47</span> remaining
          </span>
        </div>
        <Link
          href={`/grade/${assignmentId}/sub-001`}
          className="rounded-xl px-4 py-2 text-sm font-semibold transition-all hover:opacity-80"
          style={{ background: '#234e32', color: '#ffffff' }}
        >
          Start grading →
        </Link>
      </div>
    </div>
  )
}
