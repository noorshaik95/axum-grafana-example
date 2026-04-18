'use client'

import Link from 'next/link'
import { AlertTriangle, ArrowRight, TrendingUp, Users, Server, Activity } from 'lucide-react'

const SPARKLINE_DATA = [
  18, 24, 20, 35, 28, 42, 38, 50, 45, 55, 48, 60, 54, 70, 62, 75, 68, 80, 72, 85, 78, 90, 84, 95,
  88, 100, 92, 98, 95, 100,
]

const LIVE_FEED = [
  { icon: '🎓', desc: 'Stanford synced 1,247 grade submissions', time: '2m ago', color: '#d97757' },
  { icon: '👤', desc: 'New faculty account created at MIT', time: '5m ago', color: '#3e7d4f' },
  { icon: '📄', desc: 'Course "CS101" published at Berkeley', time: '11m ago', color: '#3e7d4f' },
  { icon: '⚠️', desc: 'Canvas webhook timeout — Harvard', time: '18m ago', color: '#ffb648' },
  { icon: '🏫', desc: 'Yale completed tenant provisioning', time: '24m ago', color: '#3e7d4f' },
  { icon: '💳', desc: 'Oxford renewed contract — Enterprise', time: '31m ago', color: '#3e7d4f' },
  { icon: '🔒', desc: 'MFA reset for admin@cornell.edu', time: '45m ago', color: '#6a6e62' },
  { icon: '📊', desc: 'Metrics report generated for Q1 2026', time: '1h ago', color: '#6a6e62' },
]

export default function OpsPage() {
  const maxSparkline = Math.max(...SPARKLINE_DATA)

  return (
    <div className="grid grid-cols-3 gap-6">
      {/* Left column (2/3) */}
      <div className="col-span-2 space-y-6">
        {/* Action Hero */}
        <div className="rounded-2xl bg-gradient-to-br from-[#234e32] to-[#0f2617] p-6 text-white relative overflow-hidden">
          <div
            className="absolute inset-0 opacity-10"
            style={{
              backgroundImage: 'radial-gradient(circle at 80% 20%, #5d9a6c 0%, transparent 50%)',
            }}
          />
          <div className="relative">
            <p className="text-xs font-medium tracking-widest text-[#ffb648] uppercase mb-2">
              ● P1 · Stanford
            </p>
            <h1
              className="text-2xl font-bold mb-1"
              style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
            >
              Grade sync backlog — 1,247 submissions
            </h1>
            <p className="text-white/60 text-sm mb-5">
              ETA to clear: ~14 min · Kafka consumer lag on grading-queue
            </p>
            <Link
              href="/incidents/1"
              className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-[#ffb648] text-[#12170f] text-sm font-semibold hover:bg-[#ffc76b] transition-colors"
            >
              Open incident
              <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </div>

        {/* Stat boxes */}
        <div className="grid grid-cols-4 gap-4">
          {[
            { label: 'Schools', value: '42', icon: Server, color: '#3e7d4f', bg: '#dde9df' },
            { label: 'MAU', value: '128k', icon: Users, color: '#234e32', bg: '#dde9df' },
            { label: 'Uptime', value: '99.97%', icon: Activity, color: '#234e32', bg: '#dde9df' },
            {
              label: 'Incidents',
              value: '1 P1',
              icon: AlertTriangle,
              color: '#d97757',
              bg: '#fde8e0',
            },
          ].map((s) => (
            <div key={s.label} className="rounded-xl border border-[#e4e0d4] bg-white p-4">
              <div
                className="flex h-8 w-8 items-center justify-center rounded-lg mb-3"
                style={{ backgroundColor: s.bg }}
              >
                <s.icon className="h-4 w-4" style={{ color: s.color }} />
              </div>
              <p className="text-xl font-bold text-[#12170f]">{s.value}</p>
              <p className="text-xs text-[#6a6e62] mt-0.5">{s.label}</p>
            </div>
          ))}
        </div>

        {/* Signups sparkline */}
        <div className="rounded-xl border border-[#e4e0d4] bg-white p-5">
          <div className="flex items-center justify-between mb-4">
            <div>
              <p className="text-sm font-semibold text-[#12170f]">New signups</p>
              <p className="text-xs text-[#6a6e62]">Last 30 days</p>
            </div>
            <div className="flex items-center gap-1 text-[#3e7d4f] text-xs font-medium">
              <TrendingUp className="h-3.5 w-3.5" />
              +18% vs last month
            </div>
          </div>
          <div className="flex items-end gap-1 h-16">
            {SPARKLINE_DATA.map((v, i) => (
              <div
                key={i}
                className="flex-1 rounded-sm transition-all"
                style={{
                  height: `${(v / maxSparkline) * 100}%`,
                  backgroundColor: i === SPARKLINE_DATA.length - 1 ? '#ffb648' : '#5d9a6c',
                  opacity: 0.7 + (i / SPARKLINE_DATA.length) * 0.3,
                }}
              />
            ))}
          </div>
        </div>
      </div>

      {/* Right column (1/3) */}
      <div className="col-span-1">
        <div className="rounded-xl border border-[#e4e0d4] bg-white h-full p-5">
          <p className="text-sm font-semibold text-[#12170f] mb-4">Live activity</p>
          <div className="space-y-3">
            {LIVE_FEED.map((item, i) => (
              <div key={i} className="flex items-start gap-3">
                <span className="text-base shrink-0 mt-0.5">{item.icon}</span>
                <div className="min-w-0 flex-1">
                  <p className="text-xs text-[#12170f] leading-snug">{item.desc}</p>
                  <p className="text-[10px] text-[#6a6e62] mt-0.5">{item.time}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
