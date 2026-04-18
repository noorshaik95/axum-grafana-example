'use client'

import Link from 'next/link'
import { TrendingUp, CheckSquare, BarChart2, AlertTriangle, ArrowRight } from 'lucide-react'

const struggleTopics = [
  { topic: 'Tail recursion', pct: 68, students: 32 },
  { topic: 'Type inference', pct: 52, students: 24 },
  { topic: 'Monadic bind', pct: 44, students: 21 },
  { topic: 'Pattern matching', pct: 28, students: 13 },
  { topic: 'List comprehensions', pct: 18, students: 9 },
]

export default function AnalyticsPage() {
  const maxPct = Math.max(...struggleTopics.map((t) => t.pct))

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="font-serif text-2xl text-[#12170f]">Analytics</h1>
        <p className="text-sm mt-1" style={{ color: '#6a6e62' }}>
          CS 3110 · Spring 2026 · Week 7
        </p>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-3 gap-3">
        {[
          {
            label: 'Engagement',
            value: '82%',
            sub: '+4% vs last week',
            Icon: TrendingUp,
            color: '#234e32',
            bg: '#f2f7f3',
          },
          {
            label: 'Completion',
            value: '71%',
            sub: 'modules finished avg',
            Icon: CheckSquare,
            color: '#234e32',
            bg: '#f2f7f3',
          },
          {
            label: 'Median grade',
            value: '78%',
            sub: 'across all assignments',
            Icon: BarChart2,
            color: '#234e32',
            bg: '#f2f7f3',
          },
        ].map((stat) => (
          <div
            key={stat.label}
            className="rounded-xl border p-5"
            style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
          >
            <div
              className="flex h-10 w-10 items-center justify-center rounded-lg mb-3"
              style={{ background: stat.bg }}
            >
              <stat.Icon className="h-5 w-5" style={{ color: stat.color }} />
            </div>
            <p className="font-serif text-2xl font-bold" style={{ color: stat.color }}>
              {stat.value}
            </p>
            <p className="text-sm font-medium text-[#12170f] mt-0.5">{stat.label}</p>
            <p className="text-xs mt-0.5" style={{ color: '#6a6e62' }}>
              {stat.sub}
            </p>
          </div>
        ))}
      </div>

      {/* Where students struggle */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div
          className="px-5 py-3 border-b"
          style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
        >
          <h2 className="text-sm font-semibold text-[#12170f]">Where students struggle</h2>
          <p className="text-xs mt-0.5" style={{ color: '#6a6e62' }}>
            % of students getting below 70% on topics
          </p>
        </div>
        <div className="p-5 space-y-3">
          {struggleTopics.map((topic) => {
            const barWidth = (topic.pct / maxPct) * 100
            const barColor = topic.pct >= 60 ? '#d97757' : topic.pct >= 40 ? '#ffb648' : '#3e7d4f'
            return (
              <div key={topic.topic}>
                <div className="flex items-center justify-between mb-1.5">
                  <span className="text-sm text-[#12170f]">{topic.topic}</span>
                  <div className="flex items-center gap-2">
                    <span className="text-xs" style={{ color: '#6a6e62' }}>
                      {topic.students} students
                    </span>
                    <span className="font-mono text-sm font-semibold" style={{ color: barColor }}>
                      {topic.pct}%
                    </span>
                  </div>
                </div>
                <div className="h-2 rounded-full overflow-hidden" style={{ background: '#f6f3ec' }}>
                  <div
                    className="h-full rounded-full transition-all"
                    style={{ width: `${barWidth}%`, background: barColor }}
                  />
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Actionable card */}
      <div
        className="rounded-xl border px-5 py-5"
        style={{
          background: 'linear-gradient(135deg, #f2f7f3 0%, #dde9df 100%)',
          borderColor: '#b8d2bd',
        }}
      >
        <div className="flex items-start gap-3">
          <div
            className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl mt-0.5"
            style={{ background: '#234e32' }}
          >
            <AlertTriangle className="h-5 w-5 text-white" />
          </div>
          <div className="flex-1">
            <h3 className="font-semibold text-[#12170f] mb-1">Suggested action</h3>
            <p className="text-sm leading-relaxed" style={{ color: '#234e32' }}>
              68% of students are struggling with <strong>tail recursion</strong>. Consider adding a
              live demo in Lecture 8, or creating a worked example in the course materials.
              Batch-grading PS4 with shared feedback is also a high-leverage action.
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2 mt-4">
          <Link
            href="/grade/ps4"
            className="flex items-center gap-1.5 rounded-xl px-4 py-2 text-sm font-semibold transition-all hover:opacity-80"
            style={{ background: '#234e32', color: '#ffffff' }}
          >
            Open grading queue
            <ArrowRight className="h-4 w-4" />
          </Link>
          <Link
            href="/courses/cs3110/content"
            className="flex items-center gap-1.5 rounded-xl border px-4 py-2 text-sm font-medium transition-colors hover:bg-white/50"
            style={{ borderColor: '#8ab694', color: '#234e32' }}
          >
            Add course material
          </Link>
        </div>
      </div>

      {/* At-risk alert */}
      <div
        className="rounded-xl border px-5 py-4 flex items-center gap-4"
        style={{ background: 'rgba(217,119,87,0.06)', borderColor: 'rgba(217,119,87,0.3)' }}
      >
        <AlertTriangle className="h-5 w-5 shrink-0" style={{ color: '#d97757' }} />
        <div className="flex-1">
          <p className="text-sm font-semibold text-[#12170f]">2 students at risk</p>
          <p className="text-xs mt-0.5" style={{ color: '#6a6e62' }}>
            Alex Morrison and Jordan Kim haven't submitted in 2+ weeks.
          </p>
        </div>
        <Link
          href="/roster"
          className="shrink-0 rounded-xl px-3 py-1.5 text-xs font-semibold transition-all hover:opacity-80"
          style={{ background: '#d97757', color: '#ffffff' }}
        >
          View roster
        </Link>
      </div>
    </div>
  )
}
