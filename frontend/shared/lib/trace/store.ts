/**
 * Shared trace store.
 *
 * The API client writes the current request's `X-Request-ID`, traceparent,
 * route, and HTTP status into this store on every response. The dev-only
 * <TraceWidget /> subscribes so engineers can copy the correlation ID and
 * jump straight into Grafana Tempo.
 *
 * Why `useSyncExternalStore` instead of zustand? The shared package must
 * stay dependency-free (it's consumed by three portals). React's built-in
 * external store API gives us subscriptions + concurrent-mode safety
 * without adding a new peer dep.
 */

import { useSyncExternalStore } from 'react'

export interface TraceEntry {
  requestId: string
  traceparent: string | null
  /** `${method} ${url}` captured from the fetch call. */
  route: string
  status: number
  /** epoch ms when the response returned. */
  ts: number
  /** ms spent in-flight (request start → response headers). */
  durationMs: number
}

export interface TraceStoreState {
  current: TraceEntry | null
  history: TraceEntry[]
}

const INITIAL: TraceStoreState = { current: null, history: [] }
const MAX_HISTORY = 25

type Listener = () => void

class TraceStore {
  private state: TraceStoreState = INITIAL
  private listeners = new Set<Listener>()

  getSnapshot = (): TraceStoreState => this.state
  getServerSnapshot = (): TraceStoreState => INITIAL

  subscribe = (l: Listener) => {
    this.listeners.add(l)
    return () => {
      this.listeners.delete(l)
    }
  }

  record(entry: TraceEntry) {
    const history = [entry, ...this.state.history].slice(0, MAX_HISTORY)
    this.state = { current: entry, history }
    this.emit()
  }

  clear() {
    this.state = INITIAL
    this.emit()
  }

  private emit() {
    this.listeners.forEach((l) => l())
  }
}

export const traceStore = new TraceStore()

export function useTraceStore(): TraceStoreState {
  return useSyncExternalStore(
    traceStore.subscribe,
    traceStore.getSnapshot,
    traceStore.getServerSnapshot
  )
}

export function useCurrentTrace(): TraceEntry | null {
  return useTraceStore().current
}
