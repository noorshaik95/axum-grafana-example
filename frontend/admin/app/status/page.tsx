'use client'

import { ExternalLink } from 'lucide-react'

type ServiceStatus = {
  name: string
  latency: string
  metric: string
  health: 'healthy' | 'degraded' | 'down'
}

const SERVICES: ServiceStatus[] = [
  { name: 'API Gateway', latency: '24ms', metric: 'p99: 48ms', health: 'healthy' },
  { name: 'Web App', latency: '185ms', metric: 'TTFB avg', health: 'healthy' },
  { name: 'Canvas Webhooks', latency: '1,240ms', metric: '11% timeout rate', health: 'degraded' },
  { name: 'Auth & SSO', latency: '18ms', metric: 'p99: 35ms', health: 'healthy' },
  { name: 'Video CDN', latency: '320ms', metric: 'EU elevated', health: 'degraded' },
  { name: 'Search', latency: '42ms', metric: 'p99: 80ms', health: 'healthy' },
  { name: 'Grading Queue', latency: '—', metric: 'Kafka lag: 1,247', health: 'down' },
  { name: 'Email Service', latency: '95ms', metric: 'delivery rate 99.1%', health: 'healthy' },
]

const HEALTH_CONFIG = {
  healthy: { dot: '#3e7d4f', label: 'Operational', bg: 'bg-[#dde9df]', text: 'text-[#234e32]' },
  degraded: { dot: '#d97757', label: 'Degraded', bg: 'bg-[#fde8e0]', text: 'text-[#d97757]' },
  down: { dot: '#ef4444', label: 'Down', bg: 'bg-[#fee2e2]', text: 'text-[#dc2626]' },
}

export default function StatusPage() {
  const allHealthy = SERVICES.every((s) => s.health === 'healthy')
  const hasDown = SERVICES.some((s) => s.health === 'down')
  const hasDegraded = SERVICES.some((s) => s.health === 'degraded')

  const overallLabel = hasDown
    ? 'Partial outage'
    : hasDegraded
      ? 'Degraded performance'
      : 'All systems operational'
  const overallColor = hasDown ? '#ef4444' : hasDegraded ? '#d97757' : '#3e7d4f'

  return (
    <div className="space-y-6 max-w-2xl">
      <div className="flex items-start justify-between">
        <div>
          <h1
            className="text-2xl font-bold text-[#12170f]"
            style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
          >
            Service Status
          </h1>
          <p className="text-xs text-[#6a6e62] mt-1">Internal · public page mirrors this</p>
        </div>
        <a href="#" className="flex items-center gap-1 text-xs text-[#3e7d4f] hover:underline">
          Public page <ExternalLink className="h-3 w-3" />
        </a>
      </div>

      {/* Overall status */}
      <div
        className="rounded-xl p-4 flex items-center gap-3"
        style={{ backgroundColor: overallColor + '14', border: `1px solid ${overallColor}40` }}
      >
        <span className="h-3 w-3 rounded-full shrink-0" style={{ backgroundColor: overallColor }} />
        <p className="text-sm font-semibold" style={{ color: overallColor }}>
          {overallLabel}
        </p>
        <p className="text-xs text-[#6a6e62] ml-auto">Updated just now</p>
      </div>

      {/* Service rows */}
      <div className="rounded-xl border border-[#e4e0d4] bg-white overflow-hidden">
        {SERVICES.map((svc, i) => {
          const cfg = HEALTH_CONFIG[svc.health]
          return (
            <div
              key={svc.name}
              className={`flex items-center justify-between px-5 py-3.5 ${
                i < SERVICES.length - 1 ? 'border-b border-[#e4e0d4]' : ''
              }`}
            >
              <div className="flex items-center gap-3">
                <span
                  className="h-2 w-2 rounded-full shrink-0"
                  style={{ backgroundColor: cfg.dot }}
                />
                <p className="text-sm font-medium text-[#12170f]">{svc.name}</p>
              </div>
              <div className="flex items-center gap-4">
                <p className="text-xs text-[#6a6e62] font-mono">{svc.latency}</p>
                <p className="text-xs text-[#6a6e62]">{svc.metric}</p>
                <span
                  className={`text-xs px-2 py-0.5 rounded-full font-medium ${cfg.bg} ${cfg.text}`}
                >
                  {cfg.label}
                </span>
              </div>
            </div>
          )
        })}
      </div>

      <p className="text-xs text-[#6a6e62] text-center">
        Checked every 30 seconds · Last 90 days uptime: 99.97%
      </p>
    </div>
  )
}
