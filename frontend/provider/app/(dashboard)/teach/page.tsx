'use client'

import Link from 'next/link'
import { AlertTriangle, Clock, MessageSquare, ArrowRight, ChevronRight } from 'lucide-react'

const todayTasks = [
  {
    id: 1,
    icon: Clock,
    label: 'Lecture 7',
    detail: '10:00 – 11:15am · Room 204',
    status: 'upcoming',
    href: '/lecture/live',
    color: '#234e32',
  },
  {
    id: 2,
    icon: MessageSquare,
    label: 'Threads needing reply',
    detail: '6 unanswered · oldest 14h ago',
    status: 'urgent',
    href: '/discussion',
    color: '#d97757',
  },
  {
    id: 3,
    icon: Clock,
    label: 'Office hours',
    detail: '2:00 – 3:00pm · 4 students booked',
    status: 'upcoming',
    href: '/office-hours',
    color: '#234e32',
  },
  {
    id: 4,
    icon: AlertTriangle,
    label: 'At-risk students',
    detail: '2 students flagged this week',
    status: 'alert',
    href: '/roster',
    color: '#d97757',
  },
]

export default function TeachPage() {
  return (
    <div className="space-y-6">
      {/* Action Hero */}
      <div
        className="relative rounded-2xl overflow-hidden px-8 py-10"
        style={{
          background: 'linear-gradient(135deg, #0f2617 0%, #1a3a26 40%, #234e32 100%)',
        }}
      >
        {/* Ambient amber highlight */}
        <div
          className="absolute -top-20 -right-20 h-64 w-64 rounded-full opacity-20 blur-3xl"
          style={{ background: '#ffb648' }}
        />

        <div className="relative">
          {/* Eyebrow */}
          <div className="flex items-center gap-2 mb-3">
            <span
              className="inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-mono font-medium"
              style={{ background: 'rgba(255,182,72,0.2)', color: '#ffb648' }}
            >
              <span className="h-1.5 w-1.5 rounded-full bg-current animate-pulse" />
              47 in queue · PS4
            </span>
          </div>

          {/* Heading */}
          <h1 className="font-serif text-3xl text-white mb-2 leading-tight">
            Grade 47 submissions before Lecture 7
          </h1>
          <p className="text-base mb-6" style={{ color: 'rgba(255,255,255,0.65)' }}>
            Batch-grade tail-recursion Q3s together. ~28 min.
          </p>

          {/* CTA */}
          <Link
            href="/grade/ps4"
            className="inline-flex items-center gap-2 rounded-xl px-5 py-3 text-sm font-semibold transition-all hover:scale-105"
            style={{ background: '#ffb648', color: '#12170f' }}
          >
            Open queue
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </div>

      {/* Today's task list */}
      <div>
        <h2 className="font-serif text-xl text-[#12170f] mb-3">Today</h2>
        <div className="space-y-2">
          {todayTasks.map((task) => {
            const Icon = task.icon
            const isUrgent = task.status === 'urgent' || task.status === 'alert'
            return (
              <Link
                key={task.id}
                href={task.href}
                className="flex items-center gap-4 rounded-xl px-4 py-3.5 border group transition-all hover:shadow-sm"
                style={{
                  background: isUrgent ? 'rgba(217,119,87,0.06)' : '#ffffff',
                  borderColor: isUrgent ? 'rgba(217,119,87,0.3)' : '#e4e0d4',
                }}
              >
                <div
                  className="flex h-9 w-9 items-center justify-center rounded-lg shrink-0"
                  style={{
                    background: isUrgent ? 'rgba(217,119,87,0.12)' : '#f2f7f3',
                  }}
                >
                  <Icon className="h-4 w-4" style={{ color: isUrgent ? '#d97757' : '#234e32' }} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-[#12170f]">{task.label}</p>
                  <p className="text-xs mt-0.5" style={{ color: '#6a6e62' }}>
                    {task.detail}
                  </p>
                </div>
                <ChevronRight
                  className="h-4 w-4 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
                  style={{ color: '#6a6e62' }}
                />
              </Link>
            )
          })}
        </div>
      </div>

      {/* Quick stats row */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {[
          { label: 'Students', value: '47', sub: 'enrolled', color: '#234e32' },
          { label: 'At risk', value: '2', sub: 'flagged', color: '#d97757' },
          { label: 'Median grade', value: '78%', sub: 'PS3 avg', color: '#234e32' },
          { label: 'Pulse', value: '82%', sub: 'engagement', color: '#234e32' },
        ].map((stat) => (
          <div
            key={stat.label}
            className="rounded-xl border p-4"
            style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
          >
            <p className="text-2xl font-serif font-bold" style={{ color: stat.color }}>
              {stat.value}
            </p>
            <p className="text-xs font-medium text-[#12170f] mt-0.5">{stat.label}</p>
            <p className="text-xs mt-0.5" style={{ color: '#6a6e62' }}>
              {stat.sub}
            </p>
          </div>
        ))}
      </div>
    </div>
  )
}
