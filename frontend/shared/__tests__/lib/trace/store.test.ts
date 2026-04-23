import { traceStore, type TraceEntry } from '../../../lib/trace/store'

function sampleEntry(overrides: Partial<TraceEntry> = {}): TraceEntry {
  return {
    requestId: 'req-1',
    traceparent: '00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01',
    route: 'GET /api/health',
    status: 200,
    ts: Date.now(),
    durationMs: 12,
    ...overrides,
  }
}

describe('traceStore', () => {
  beforeEach(() => traceStore.clear())

  it('starts empty and updates on record()', () => {
    expect(traceStore.getSnapshot().current).toBeNull()

    const entry = sampleEntry()
    traceStore.record(entry)

    expect(traceStore.getSnapshot().current).toEqual(entry)
    expect(traceStore.getSnapshot().history).toHaveLength(1)
  })

  it('keeps history ordered newest-first and caps at 25', () => {
    for (let i = 0; i < 40; i += 1) {
      traceStore.record(sampleEntry({ requestId: `req-${i}` }))
    }
    const snap = traceStore.getSnapshot()
    expect(snap.history).toHaveLength(25)
    expect(snap.history[0].requestId).toBe('req-39')
    expect(snap.current?.requestId).toBe('req-39')
  })

  it('notifies subscribers on record and clear', () => {
    const spy = jest.fn()
    const unsub = traceStore.subscribe(spy)

    traceStore.record(sampleEntry())
    traceStore.record(sampleEntry({ requestId: 'req-2' }))
    traceStore.clear()

    expect(spy).toHaveBeenCalledTimes(3)
    unsub()
    traceStore.record(sampleEntry({ requestId: 'req-3' }))
    expect(spy).toHaveBeenCalledTimes(3)
  })
})
