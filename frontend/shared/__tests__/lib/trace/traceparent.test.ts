import {
  newTraceparent,
  isValidTraceparent,
  traceIdFromTraceparent,
} from '../../../lib/trace/traceparent'

describe('traceparent helpers', () => {
  it('generates a valid W3C traceparent with sampled flag', () => {
    for (let i = 0; i < 32; i += 1) {
      const tp = newTraceparent()
      expect(isValidTraceparent(tp)).toBe(true)
      expect(tp.endsWith('-01')).toBe(true)
    }
  })

  it('rejects malformed traceparents', () => {
    expect(isValidTraceparent('')).toBe(false)
    expect(isValidTraceparent(null)).toBe(false)
    expect(isValidTraceparent(undefined)).toBe(false)
    expect(isValidTraceparent('01-abc-def-ff')).toBe(false)
    expect(isValidTraceparent('00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-0q')).toBe(
      false
    )
    // all-zero trace id
    expect(isValidTraceparent('00-00000000000000000000000000000000-00f067aa0ba902b7-01')).toBe(
      false
    )
  })

  it('extracts the 32-char trace id for Tempo deep links', () => {
    const tp = '00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01'
    expect(traceIdFromTraceparent(tp)).toBe('4bf92f3577b34da6a3ce929d0e0e4736')
    expect(traceIdFromTraceparent('invalid')).toBeNull()
  })
})
