'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, ArrowUp, Bell, MessageSquare } from 'lucide-react'

const MOCK_INCIDENTS: Record<
  string,
  {
    title: string
    school: string
    priority: string
    status: string
    impacted: number
    description: string
    timeline: Array<{ time: string; event: string; author?: string }>
  }
> = {
  '1': {
    title: 'Grade sync backlog — 1,247 submissions',
    school: 'Stanford',
    priority: 'P1',
    status: 'open',
    impacted: 1247,
    description:
      'Kafka consumer lag on grading-queue topic is preventing grade submissions from being processed. Students are seeing pending grades in the portal.',
    timeline: [
      { time: '14:32:08', event: 'Incident created automatically via Kafka lag alert' },
      { time: '14:33:15', event: 'On-call engineer notified via PagerDuty', author: 'system' },
      {
        time: '14:41:02',
        event: 'Root cause identified: consumer group rebalance loop',
        author: 'alex@slate.io',
      },
      {
        time: '14:55:30',
        event: 'Fix deployed — consumer group reset initiated',
        author: 'alex@slate.io',
      },
      { time: '15:02:44', event: 'Processing resumed — 312 submissions cleared', author: 'system' },
    ],
  },
  '2': {
    title: 'Canvas webhook timeout intermittent',
    school: 'Harvard',
    priority: 'P3',
    status: 'watch',
    impacted: 85,
    description:
      'Canvas webhook delivery to Slate is timing out intermittently (10–15% failure rate). Instructors may see delayed course sync.',
    timeline: [
      { time: '08:14:22', event: 'Alert triggered: webhook error rate > 5%' },
      {
        time: '08:20:00',
        event: 'Investigating Canvas API status page',
        author: 'morgan@slate.io',
      },
      {
        time: '09:05:15',
        event: 'Canvas reported degraded webhooks in their status page',
        author: 'system',
      },
    ],
  },
}

export default function IncidentRoomPage() {
  const params = useParams()
  const id = typeof params.id === 'string' ? params.id : '1'
  const inc = MOCK_INCIDENTS[id] ?? MOCK_INCIDENTS['1']

  return (
    <div className="space-y-6 max-w-2xl">
      <Link
        href="/incidents"
        className="inline-flex items-center gap-1 text-sm text-[#6a6e62] hover:text-[#234e32]"
      >
        <ArrowLeft className="h-3.5 w-3.5" /> Back to incidents
      </Link>

      {/* Header */}
      <div>
        <div className="flex items-center gap-2 mb-2">
          <span
            className={`text-xs font-bold px-2 py-0.5 rounded ${
              inc.priority === 'P1' ? 'bg-[#d97757] text-white' : 'bg-[#ffb648]/30 text-[#856404]'
            }`}
          >
            {inc.priority}
          </span>
          <span className="text-xs text-[#6a6e62]">{inc.school}</span>
          <span
            className={`text-xs px-2 py-0.5 rounded-full ${
              inc.status === 'open' ? 'bg-[#fde8e0] text-[#d97757]' : 'bg-[#dde9df] text-[#234e32]'
            }`}
          >
            {inc.status}
          </span>
        </div>
        <h1
          className="text-2xl font-bold text-[#12170f]"
          style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
        >
          {inc.title}
        </h1>
      </div>

      {/* Impact card */}
      <div className="rounded-xl bg-[#12170f] text-white p-5">
        <p className="text-3xl font-bold mb-1">{inc.impacted.toLocaleString()}</p>
        <p className="text-white/50 text-xs uppercase tracking-wide mb-2">affected submissions</p>
        <p className="text-sm text-white/80 leading-relaxed">{inc.description}</p>
      </div>

      {/* Timeline */}
      <div className="rounded-xl border border-[#e4e0d4] bg-white p-5">
        <p className="text-sm font-semibold text-[#12170f] mb-4">Timeline</p>
        <div className="space-y-3">
          {inc.timeline.map((t, i) => (
            <div key={i} className="flex items-start gap-3">
              <span
                className="shrink-0 text-xs mt-0.5 font-mono text-[#6a6e62]"
                style={{ fontFamily: 'JetBrains Mono, monospace' }}
              >
                {t.time}
              </span>
              <div className="flex-1">
                <p className="text-sm text-[#12170f]">{t.event}</p>
                {t.author && <p className="text-xs text-[#6a6e62]">— {t.author}</p>}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-3 flex-wrap">
        <button className="flex items-center gap-2 px-4 py-2 rounded-lg bg-[#234e32] text-white text-sm font-medium hover:bg-[#1a3a26] transition-colors">
          <MessageSquare className="h-4 w-4" />
          Post update
        </button>
        <button className="flex items-center gap-2 px-4 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm font-medium text-[#12170f] hover:bg-[#f6f3ec] transition-colors">
          <Bell className="h-4 w-4" />
          Notify tenant
        </button>
        <button className="flex items-center gap-2 px-4 py-2 rounded-lg bg-[#d97757] text-white text-sm font-medium hover:bg-[#c4694a] transition-colors">
          <ArrowUp className="h-4 w-4" />
          Escalate P0
        </button>
      </div>
    </div>
  )
}
