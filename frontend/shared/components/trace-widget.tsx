'use client'

import * as React from 'react'
import { cn } from '../utils/index'
import { useCurrentTrace, useTraceStore } from '../lib/trace/store'
import { traceIdFromTraceparent } from '../lib/trace/traceparent'

export interface TraceWidgetProps {
  /** Explicit dev-mode override. Defaults to `process.env.NODE_ENV !== 'production'`. */
  enabled?: boolean
  /** Grafana base URL — used to construct the trace-explore deep link. */
  grafanaUrl?: string
  className?: string
}

/**
 * Dev-only floating widget that surfaces the current request's X-Request-ID +
 * W3C traceparent. Click copies the ID to the clipboard; the side link opens
 * the Grafana Tempo trace-explore view scoped to the current trace.
 *
 * Gate via `enabled`. The widget auto-disables in production builds so it
 * never ships to end users. Import once from each portal's root layout.
 */
export function TraceWidget({
  enabled,
  grafanaUrl = 'http://localhost:3200',
  className,
}: TraceWidgetProps) {
  const isEnabled =
    enabled ?? (typeof process !== 'undefined' && process.env.NODE_ENV !== 'production')
  const current = useCurrentTrace()
  const { history } = useTraceStore()
  const [open, setOpen] = React.useState(false)
  const [copied, setCopied] = React.useState(false)

  if (!isEnabled) return null
  if (!current) return null

  const traceId = traceIdFromTraceparent(current.traceparent)
  const tempoUrl = traceId
    ? `${grafanaUrl}/explore?left=${encodeURIComponent(
        JSON.stringify({ datasource: 'Tempo', queries: [{ query: traceId }] })
      )}`
    : null

  const copy = async (text: string) => {
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
        setCopied(true)
        setTimeout(() => setCopied(false), 1200)
      }
    } catch {
      // clipboard unavailable — silently ignore
    }
  }

  const statusTone =
    current.status >= 500
      ? 'bg-red-500 text-white'
      : current.status >= 400
        ? 'bg-amber-500 text-white'
        : 'bg-forest-600 text-white'

  return (
    <div
      data-testid="trace-widget"
      className={cn(
        'pointer-events-auto fixed bottom-3 right-3 z-[60] select-text',
        'font-mono text-[11px] leading-tight text-warm-900',
        className
      )}
    >
      <div className="rounded-card border border-warm-200 bg-cream/95 shadow-md backdrop-blur">
        <div className="flex items-center gap-2 px-3 py-2">
          <span
            className={cn(
              'inline-flex h-5 min-w-[2.5rem] items-center justify-center rounded-full px-2 text-[10px] font-semibold',
              statusTone
            )}
          >
            {current.status}
          </span>
          <button
            type="button"
            onClick={() => copy(current.requestId)}
            title="Click to copy X-Request-ID"
            className="rounded px-1 py-0.5 hover:bg-warm-100"
          >
            <span className="text-warm-700">req </span>
            <span className="text-forest-800">{current.requestId.slice(0, 8)}…</span>
          </button>
          {tempoUrl ? (
            <a
              href={tempoUrl}
              target="_blank"
              rel="noreferrer"
              title="Open trace in Grafana Tempo"
              className="rounded px-1 py-0.5 text-forest-700 hover:bg-forest-50"
            >
              tempo ↗
            </a>
          ) : null}
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            className="ml-1 rounded px-1.5 py-0.5 text-warm-700 hover:bg-warm-100"
          >
            {open ? '×' : `+${history.length}`}
          </button>
        </div>
        {copied ? <div className="px-3 pb-2 text-[10px] text-forest-700">copied ✓</div> : null}
        {open ? (
          <div className="max-h-64 w-80 overflow-y-auto border-t border-warm-200/80 px-3 py-2">
            <div className="mb-1 text-[10px] uppercase tracking-wide text-warm-700">
              Recent requests
            </div>
            <ul className="space-y-1">
              {history.map((e) => (
                <li
                  key={`${e.requestId}-${e.ts}`}
                  className="flex items-center justify-between gap-2"
                >
                  <span className="truncate text-warm-900">{e.route}</span>
                  <span className="shrink-0 text-warm-700">
                    {e.status} · {Math.round(e.durationMs)}ms
                  </span>
                </li>
              ))}
            </ul>
          </div>
        ) : null}
      </div>
    </div>
  )
}

TraceWidget.displayName = 'TraceWidget'
