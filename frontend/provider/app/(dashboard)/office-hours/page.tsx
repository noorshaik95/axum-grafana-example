'use client'

import { useState } from 'react'
import { Clock, User, MessageSquare, CheckCircle2, Plus } from 'lucide-react'

const bookedSlots = [
  {
    id: 1,
    time: '2:00 pm',
    student: 'Jordan Kim',
    initials: 'JK',
    question: 'Struggling with PS4 Q2 — tail recursion accumulator pattern',
    preRead: true,
    risk: 'slipping',
  },
  {
    id: 2,
    time: '2:15 pm',
    student: 'Sam Torres',
    initials: 'ST',
    question: 'General check-in, grade dropped recently',
    preRead: true,
    risk: 'slipping',
  },
  {
    id: 3,
    time: '2:30 pm',
    student: 'Casey Walsh',
    initials: 'CW',
    question: 'Question about extra credit opportunity',
    preRead: false,
    risk: 'healthy',
  },
  {
    id: 4,
    time: '2:45 pm',
    student: 'Morgan Lee',
    initials: 'ML',
    question: 'Wants feedback on PS3 before submitting PS4',
    preRead: true,
    risk: 'healthy',
  },
]

const riskColors = {
  'at-risk': { avatarBg: 'rgba(217,119,87,0.15)', avatarText: '#d97757' },
  slipping: { avatarBg: 'rgba(255,182,72,0.15)', avatarText: '#c48d1a' },
  healthy: { avatarBg: '#f2f7f3', avatarText: '#3e7d4f' },
}

export default function OfficeHoursPage() {
  const [activeSlot, setActiveSlot] = useState<number | null>(null)

  return (
    <div className="space-y-5 max-w-2xl">
      {/* Header */}
      <div>
        <h1 className="font-serif text-2xl text-[#12170f]">Today · 2–3pm · 4 booked</h1>
        <p className="text-sm mt-1" style={{ color: '#6a6e62' }}>
          Thursday office hours — Room 204 + Zoom
        </p>
      </div>

      {/* Status bar */}
      <div
        className="flex items-center gap-4 rounded-xl border px-5 py-3"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full" style={{ background: '#3e7d4f' }} />
          <span className="text-sm font-medium text-[#12170f]">Session active</span>
        </div>
        <span style={{ color: '#e4e0d4' }}>·</span>
        <span className="text-sm" style={{ color: '#6a6e62' }}>
          <Clock className="inline h-3.5 w-3.5 mr-1" />
          Ends at 3:00 pm
        </span>
        <span style={{ color: '#e4e0d4' }}>·</span>
        <span className="text-sm" style={{ color: '#6a6e62' }}>
          4 students · 1 open slot
        </span>
      </div>

      {/* Booked slots */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div
          className="px-4 py-2.5 border-b"
          style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
        >
          <span className="text-xs font-mono font-medium" style={{ color: '#6a6e62' }}>
            BOOKED SLOTS
          </span>
        </div>
        <div className="divide-y" style={{ borderColor: '#e4e0d4' }}>
          {bookedSlots.map((slot) => {
            const colors = riskColors[slot.risk as keyof typeof riskColors]
            const isActive = activeSlot === slot.id
            return (
              <div
                key={slot.id}
                className="flex items-start gap-4 px-4 py-4 cursor-pointer transition-colors"
                style={{ background: isActive ? '#f2f7f3' : 'transparent' }}
                onClick={() => setActiveSlot(isActive ? null : slot.id)}
              >
                {/* Time */}
                <div className="shrink-0 w-16">
                  <span className="font-mono text-sm font-medium" style={{ color: '#6a6e62' }}>
                    {slot.time}
                  </span>
                </div>

                {/* Avatar */}
                <div
                  className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
                  style={{ background: colors.avatarBg, color: colors.avatarText }}
                >
                  {slot.initials}
                </div>

                {/* Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-0.5">
                    <span className="text-sm font-semibold text-[#12170f]">{slot.student}</span>
                    {slot.preRead && (
                      <span
                        className="flex items-center gap-1 text-xs rounded-full px-2 py-0.5"
                        style={{ background: '#dde9df', color: '#234e32' }}
                      >
                        <CheckCircle2 className="h-3 w-3" />
                        pre-read ✓
                      </span>
                    )}
                  </div>
                  <p className="text-xs leading-snug" style={{ color: '#6a6e62' }}>
                    {slot.question}
                  </p>
                </div>

                {/* Open button */}
                <button
                  className="shrink-0 rounded-xl px-3 py-1.5 text-xs font-medium transition-all hover:opacity-80"
                  style={{ background: '#234e32', color: '#ffffff' }}
                >
                  Open
                </button>
              </div>
            )
          })}

          {/* Open slot */}
          <div
            className="flex items-center gap-4 px-4 py-4 border-t"
            style={{ borderColor: '#e4e0d4' }}
          >
            <div className="shrink-0 w-16">
              <span className="font-mono text-sm" style={{ color: '#6a6e62' }}>
                3:00 pm
              </span>
            </div>
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border-2 border-dashed"
              style={{ borderColor: '#e4e0d4' }}
            >
              <Plus className="h-4 w-4" style={{ color: '#6a6e62' }} />
            </div>
            <div className="flex-1">
              <span className="text-sm" style={{ color: '#6a6e62' }}>
                Open slot
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Quick actions */}
      <div className="flex gap-2">
        <button
          className="flex-1 flex items-center justify-center gap-2 rounded-xl border py-3 text-sm font-medium transition-colors hover:bg-[#f6f3ec]"
          style={{ borderColor: '#e4e0d4', color: '#12170f' }}
        >
          <MessageSquare className="h-4 w-4" />
          Message all attendees
        </button>
        <button
          className="flex items-center gap-2 rounded-xl py-3 px-5 text-sm font-semibold transition-all hover:opacity-80"
          style={{ background: '#d97757', color: '#ffffff' }}
        >
          End session
        </button>
      </div>
    </div>
  )
}
