/**
 * W3C Trace Context v00 — client-side generator + parser.
 *
 * The gateway will replace our generated trace ID with its own if it receives
 * a bad one, but under happy-path the IDs flow unchanged from browser →
 * gateway → downstream services, so a Playwright click in CI can assert a
 * single trace spans multiple services.
 */

const TRACEPARENT_REGEX = /^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$/

function randomHex(bytes: number): string {
  const arr = new Uint8Array(bytes)
  if (typeof globalThis.crypto?.getRandomValues === 'function') {
    globalThis.crypto.getRandomValues(arr)
  } else {
    for (let i = 0; i < bytes; i += 1) {
      arr[i] = Math.floor(Math.random() * 256)
    }
  }
  let out = ''
  for (let i = 0; i < arr.length; i += 1) {
    out += arr[i].toString(16).padStart(2, '0')
  }
  return out
}

/**
 * Build a fresh W3C traceparent with the sampled flag on.
 * Format: `00-<32 hex trace id>-<16 hex span id>-01`.
 */
export function newTraceparent(): string {
  return `00-${randomHex(16)}-${randomHex(8)}-01`
}

export function isValidTraceparent(s: string | null | undefined): boolean {
  if (!s) return false
  if (!TRACEPARENT_REGEX.test(s)) return false
  // reject all-zero trace id
  const traceId = s.slice(3, 35)
  return !/^0+$/.test(traceId)
}

export function traceIdFromTraceparent(s: string | null | undefined): string | null {
  if (!isValidTraceparent(s)) return null
  return (s as string).slice(3, 35)
}
