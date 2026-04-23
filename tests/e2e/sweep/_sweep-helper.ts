/**
 * Integration sweep helper (task #40).
 *
 * Drives a logged-in browser through a list of routes, capturing:
 *   - page HTTP status
 *   - every /api/* XHR (method, path, status)
 *   - JS console errors
 *   - nav container presence (sidebar / top-bar)
 *   - simple "real data" heuristic (not loading-forever, not mock placeholder)
 *
 * Writes one JSON file per portal to tests/e2e/test-results/sweep-<portal>.json
 * which the top-level task aggregator turns into the markdown tables
 * requested by po-analyst.
 */

import { type Page, type Request, type Response } from '@playwright/test'
import * as fs from 'node:fs'
import * as path from 'node:path'

export type ApiCall = {
  method: string
  path: string
  status: number
}

export type RouteReport = {
  route: string
  pageHttp: number | null
  apiCalls: ApiCall[]
  apiErrors: ApiCall[]
  consoleErrors: string[]
  navPresent: boolean
  widgetsReal: boolean | null
  widgetsReason: string
  loadError: string | null
}

export type PortalReport = {
  portal: 'admin' | 'provider' | 'student'
  baseUrl: string
  loggedIn: boolean
  routes: RouteReport[]
  writes?: WriteProbeResult[]
  capturedAt: string
}

/**
 * `expectedVerdict` marks probes whose "unwired" result is intentional
 * (MVP scope-cut — backend not built; FE toast-rejects client-side so the UI
 * doesn't break). When the observed verdict matches the expected one, the
 * report still surfaces the row but flags `expectedMatch: true` so triage
 * can filter it out. Supported values:
 *   - 'wired'       : we expect gateway to route + backend to accept
 *   - 'unwired-mvp' : we expect gateway 404 "Route not found" — intentional
 */
export type WriteProbe = {
  method: 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  path: string
  body?: unknown
  expectedVerdict?: 'wired' | 'unwired-mvp'
}

export type WriteProbeResult = WriteProbe & {
  status: number
  /**
   * 'wired'       → gateway recognized the route (2xx, or a semantic 4xx body)
   * 'unwired'     → gateway returned "Route not found" (gateway 404)
   * 'auth-denied' → 401/403 — route exists but creds insufficient (still counts as wired)
   * 'server-err'  → 5xx (can't distinguish wired vs. backend-broken without inspection)
   * 'unknown'     → network error or unparseable body
   */
  verdict: 'wired' | 'unwired' | 'auth-denied' | 'server-err' | 'unknown'
  /** True iff `expectedVerdict` was set AND observed verdict matches it. */
  expectedMatch: boolean
  bodySample: string
}

const API_URL = process.env.API_URL ?? 'http://localhost:8080'

/**
 * Do a real login POST. Returns token map + role or null.
 * Writes all known localStorage keys (slate_token / admin_token / admin_auth_token)
 * + the slate_token cookie so ALL portals' auth middleware accepts us.
 */
export async function loginAdmin(): Promise<string | null> {
  // The admin-auth service has a separate endpoint
  try {
    const res = await fetch(`${API_URL}/api/admin/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'admin@slate.edu', password: 'Admin@123456' }),
    })
    if (!res.ok) return null
    const json = (await res.json()) as { accessToken?: string; access_token?: string }
    return json.accessToken ?? json.access_token ?? null
  } catch {
    return null
  }
}

export async function loginRegular(
  email = 'admin@slate.edu',
  password = 'Admin@123456'
): Promise<string | null> {
  try {
    const res = await fetch(`${API_URL}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!res.ok) return null
    const json = (await res.json()) as { accessToken?: string; access_token?: string }
    return json.accessToken ?? json.access_token ?? null
  } catch {
    return null
  }
}

export async function injectAuth(page: Page, baseUrl: string, token: string): Promise<void> {
  // Navigate to baseUrl first so localStorage + cookies are scoped to the right origin.
  await page.goto(baseUrl, { waitUntil: 'domcontentloaded' }).catch(() => {})
  await page.evaluate((tk) => {
    localStorage.setItem('slate_token', tk)
    localStorage.setItem('slate_refresh_token', tk)
    localStorage.setItem('admin_token', tk)
    localStorage.setItem('admin_auth_token', tk)
  }, token)
  await page
    .context()
    .addCookies([
      { name: 'slate_token', value: token, url: baseUrl, httpOnly: false, sameSite: 'Lax' },
    ])
}

export async function sweepRoute(
  page: Page,
  route: string,
  timeoutMs = 8000
): Promise<RouteReport> {
  const apiCalls: ApiCall[] = []
  const consoleErrors: string[] = []
  let loadError: string | null = null

  const onResponse = (resp: Response) => {
    try {
      const url = resp.url()
      if (url.includes('/api/') || url.includes(':8080')) {
        const req: Request = resp.request()
        const u = new URL(url)
        apiCalls.push({
          method: req.method(),
          path: u.pathname + (u.search || ''),
          status: resp.status(),
        })
      }
    } catch {
      /* ignore */
    }
  }
  const onConsole = (msg: any) => {
    if (msg.type() === 'error') {
      const text = String(msg.text()).slice(0, 240)
      if (!text.includes('Download the React DevTools')) consoleErrors.push(text)
    }
  }
  const onPageError = (err: Error) => {
    consoleErrors.push(`PAGEERROR: ${String(err.message).slice(0, 240)}`)
  }

  page.on('response', onResponse)
  page.on('console', onConsole)
  page.on('pageerror', onPageError)

  let pageHttp: number | null = null
  try {
    const resp = await page.goto(route, { waitUntil: 'domcontentloaded', timeout: timeoutMs })
    pageHttp = resp?.status() ?? null
    // Give XHRs a beat to fire + settle; most dashboards fire 2-4 queries at mount.
    await page
      .waitForLoadState('networkidle', { timeout: Math.max(2000, timeoutMs / 2) })
      .catch(() => {})
  } catch (e) {
    loadError = String((e as Error).message).slice(0, 240)
  }

  // Nav-present heuristic: either a <nav> element, or a sidebar data-testid,
  // or the AppShell top-bar (class name or data-slot).
  const navPresent = await page
    .evaluate(() => {
      const hasNav = !!document.querySelector('nav')
      const hasSidebar = !!document.querySelector(
        '[data-testid*="sidebar"],[data-testid*="top-bar"],[data-slot*="top-bar"],header[role="banner"],aside'
      )
      return hasNav || hasSidebar
    })
    .catch(() => false)

  // Widgets-real heuristic: the body has more than ~200 chars of non-placeholder text
  // and isn't dominated by loading spinners or explicit "Coming soon"/"Mock" strings.
  const widgetsProbe = await page
    .evaluate(() => {
      const body = document.body?.innerText ?? ''
      const text = body.replace(/\s+/g, ' ').trim()
      const len = text.length
      const lower = text.toLowerCase()
      const looksLikeMock =
        lower.includes('coming soon') ||
        lower.includes('mock data') ||
        lower.includes('not implemented') ||
        lower.includes('placeholder')
      const looksLikeLoading =
        !!document.querySelector('[data-testid*="spinner"],[class*="animate-spin"]') && len < 120
      return { len, lower, looksLikeMock, looksLikeLoading }
    })
    .catch(() => ({ len: 0, lower: '', looksLikeMock: false, looksLikeLoading: false }))

  let widgetsReal: boolean | null = null
  let widgetsReason = ''
  if (loadError) {
    widgetsReal = false
    widgetsReason = 'page failed to load'
  } else if (widgetsProbe.looksLikeMock) {
    widgetsReal = false
    widgetsReason = 'placeholder/coming-soon copy'
  } else if (widgetsProbe.looksLikeLoading) {
    widgetsReal = false
    widgetsReason = 'stuck on spinner'
  } else if (widgetsProbe.len < 100) {
    widgetsReal = false
    widgetsReason = `empty body (${widgetsProbe.len} chars)`
  } else {
    widgetsReal = true
    widgetsReason = `body=${widgetsProbe.len} chars`
  }

  page.off('response', onResponse)
  page.off('console', onConsole)
  page.off('pageerror', onPageError)

  const apiErrors = apiCalls.filter((c) => c.status >= 400)

  return {
    route,
    pageHttp,
    apiCalls,
    apiErrors,
    consoleErrors: Array.from(new Set(consoleErrors)).slice(0, 10),
    navPresent,
    widgetsReal,
    widgetsReason,
    loadError,
  }
}

/**
 * Exercise a list of write endpoints against the live gateway.
 *
 * Distinguishes "gateway route unwired" (the reason this exists — user found
 * POST /api/onboarding returns 404 at the gateway because the route isn't
 * registered) from "backend returned a semantic 4xx" (route IS wired; the
 * request just failed validation / not-found-by-id / auth).
 *
 * Verdict heuristic:
 *   - gateway "Route not found" body → 'unwired' (hard fail)
 *   - 2xx                             → 'wired'
 *   - 400/409/422                     → 'wired' (semantic validation)
 *   - 401/403                         → 'auth-denied' (route exists)
 *   - 404 with any non-"Route not found" body → 'wired' (semantic not-found)
 *   - 5xx                             → 'server-err' (indeterminate; note it)
 *   - anything else / network err     → 'unknown'
 */
export async function runWriteProbes(
  baseApiUrl: string,
  token: string,
  probes: WriteProbe[]
): Promise<WriteProbeResult[]> {
  const results: WriteProbeResult[] = []
  for (const p of probes) {
    let status = 0
    let bodyText = ''
    let verdict: WriteProbeResult['verdict'] = 'unknown'
    try {
      const res = await fetch(baseApiUrl + p.path, {
        method: p.method,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: p.body === undefined ? undefined : JSON.stringify(p.body),
      })
      status = res.status
      // 800-char sample: the 400-char limit truncated the critical
      // "unrecognized field X" text on proto-conversion 502s.
      bodyText = (await res.text()).slice(0, 800)

      const looksLikeGatewayRouteNotFound =
        status === 404 &&
        (bodyText.includes('"Route not found"') ||
          bodyText.includes('route not found') ||
          bodyText.includes('"status":404') ||
          bodyText.includes('ROUTE_NOT_FOUND'))

      if (status >= 200 && status < 300) verdict = 'wired'
      else if (looksLikeGatewayRouteNotFound) verdict = 'unwired'
      else if (status === 401 || status === 403) verdict = 'auth-denied'
      else if (status === 400 || status === 404 || status === 409 || status === 422)
        verdict = 'wired'
      else if (status >= 500) verdict = 'server-err'
      else verdict = 'unknown'
    } catch (e) {
      bodyText = `ERR: ${String((e as Error).message).slice(0, 400)}`
      verdict = 'unknown'
    }
    // expectedMatch:
    //   - no expectedVerdict set → false (neutral; everything is scrutinized)
    //   - expected 'wired'       → match when verdict ∈ {wired, auth-denied}
    //   - expected 'unwired-mvp' → match when verdict === 'unwired'
    let expectedMatch = false
    if (p.expectedVerdict === 'wired') {
      expectedMatch = verdict === 'wired' || verdict === 'auth-denied'
    } else if (p.expectedVerdict === 'unwired-mvp') {
      expectedMatch = verdict === 'unwired'
    }
    results.push({
      ...p,
      status,
      verdict,
      expectedMatch,
      bodySample: bodyText.replace(/\s+/g, ' ').trim(),
    })
  }
  return results
}

export function writeReport(portal: PortalReport['portal'], report: PortalReport): void {
  const outDir = path.join(__dirname, '..', 'test-results')
  try {
    fs.mkdirSync(outDir, { recursive: true })
  } catch {
    /* ignore */
  }
  const outFile = path.join(outDir, `sweep-${portal}.json`)
  fs.writeFileSync(outFile, JSON.stringify(report, null, 2))
  // eslint-disable-next-line no-console
  console.log(`[sweep] ${portal} report → ${outFile}`)
}
