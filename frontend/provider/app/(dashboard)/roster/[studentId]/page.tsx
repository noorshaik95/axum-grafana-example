'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, AlertTriangle, MessageSquare, Clock, BellOff } from 'lucide-react'

const mockStudent = {
  id: 'stu-001',
  name: 'Alex Morrison',
  initials: 'AM',
  risk: 'at-risk' as const,
  grade: '54%',
  lastSeen: '3 days ago',
  signals: [
    { label: 'Missed assignments', value: 'PS2, PS3, PS4', icon: AlertTriangle, color: '#d97757' },
    { label: 'No thread activity', value: '18 days', icon: MessageSquare, color: '#d97757' },
    { label: 'No OH booked', value: 'Never attended', icon: Clock, color: '#c48d1a' },
  ],
  suggestedMessage: `Hi Alex,\n\nI noticed you've missed the last three problem sets and haven't been active in the course. I wanted to reach out to check in — sometimes things come up and I want to make sure you have the support you need.\n\nPlease let me know if there's anything going on or if you'd like to chat during office hours. I'm available 2–3pm on Thursdays, and I'm happy to work out a catch-up plan.\n\nBest,\nProf. Chen`,
}

export default function StudentDetailPage() {
  useParams()
  const [messageSent, setMessageSent] = useState(false)
  const [snoozed, setSnoozed] = useState(false)

  const student = mockStudent

  return (
    <div className="space-y-5 max-w-2xl">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link
          href="/roster"
          className="flex items-center gap-1.5 text-sm transition-colors hover:text-[#234e32]"
          style={{ color: '#6a6e62' }}
        >
          <ArrowLeft className="h-4 w-4" />
          Roster
        </Link>
      </div>

      {/* Student header */}
      <div
        className="rounded-xl border px-6 py-5 flex items-center gap-4"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div
          className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full text-xl font-semibold"
          style={{ background: 'rgba(217,119,87,0.15)', color: '#d97757' }}
        >
          {student.initials}
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <h1 className="font-serif text-2xl text-[#12170f]">{student.name}</h1>
            <span
              className="rounded-full px-2.5 py-0.5 text-xs font-mono font-medium"
              style={{ background: 'rgba(217,119,87,0.15)', color: '#d97757' }}
            >
              at risk
            </span>
          </div>
          <div className="flex items-center gap-3 text-sm" style={{ color: '#6a6e62' }}>
            <span>
              Grade:{' '}
              <span className="font-semibold" style={{ color: '#d97757' }}>
                {student.grade}
              </span>
            </span>
            <span>·</span>
            <span>Last seen: {student.lastSeen}</span>
          </div>
        </div>
      </div>

      {/* Signals */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div
          className="px-4 py-2.5 border-b"
          style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
        >
          <span className="text-xs font-mono font-medium" style={{ color: '#6a6e62' }}>
            SIGNALS
          </span>
        </div>
        <div className="divide-y" style={{ borderColor: '#e4e0d4' }}>
          {student.signals.map((signal, i) => {
            const Icon = signal.icon
            return (
              <div key={i} className="flex items-center gap-3 px-4 py-3">
                <div
                  className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg"
                  style={{ background: signal.color + '18' }}
                >
                  <Icon className="h-4 w-4" style={{ color: signal.color }} />
                </div>
                <div className="flex-1">
                  <span className="text-sm font-medium text-[#12170f]">{signal.label}</span>
                </div>
                <span className="text-sm font-mono" style={{ color: signal.color }}>
                  {signal.value}
                </span>
              </div>
            )
          })}
        </div>
      </div>

      {/* Suggested message */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div
          className="flex items-center justify-between px-4 py-2.5 border-b"
          style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
        >
          <span className="text-xs font-mono font-medium" style={{ color: '#6a6e62' }}>
            SUGGESTED ACTION
          </span>
          <span className="text-xs" style={{ color: '#6a6e62' }}>
            Pre-written — edit before sending
          </span>
        </div>
        <pre
          className="px-5 py-4 text-sm leading-relaxed whitespace-pre-wrap font-sans"
          style={{ color: '#12170f', background: '#fbfaf5' }}
        >
          {student.suggestedMessage}
        </pre>
      </div>

      {/* Action buttons */}
      <div className="flex gap-2">
        <button
          onClick={() => setMessageSent(true)}
          className="flex-1 flex items-center justify-center gap-2 rounded-xl py-3 text-sm font-semibold transition-all hover:opacity-80"
          style={
            messageSent
              ? { background: '#dde9df', color: '#234e32' }
              : { background: '#234e32', color: '#ffffff' }
          }
          disabled={messageSent}
        >
          <MessageSquare className="h-4 w-4" />
          {messageSent ? '✓ Message sent' : 'Send message'}
        </button>
        <Link
          href="/office-hours"
          className="flex items-center gap-2 rounded-xl border px-4 py-3 text-sm font-medium transition-colors hover:bg-[#f6f3ec]"
          style={{ borderColor: '#e4e0d4', color: '#12170f' }}
        >
          <Clock className="h-4 w-4" />
          Book OH slot
        </Link>
        <button
          onClick={() => setSnoozed(!snoozed)}
          className="flex items-center gap-2 rounded-xl border px-4 py-3 text-sm font-medium transition-colors hover:bg-[#f6f3ec]"
          style={{ borderColor: '#e4e0d4', color: snoozed ? '#6a6e62' : '#12170f' }}
        >
          <BellOff className="h-4 w-4" />
          {snoozed ? 'Snoozed' : 'Snooze'}
        </button>
      </div>
    </div>
  )
}
