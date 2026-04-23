/**
 * Admin portal sweep (task #40). Enumerates every top-nav + sub-page route,
 * captures per-route HTTP / API / console / nav / widget state, and writes a
 * JSON report for po-analyst aggregation.
 */
import { test } from '@playwright/test'
import {
  injectAuth,
  loginAdmin,
  runWriteProbes,
  sweepRoute,
  writeReport,
  type PortalReport,
  type RouteReport,
  type WriteProbe,
} from '../sweep/_sweep-helper'

const BASE = 'http://localhost:3003'
const API_URL = process.env.API_URL ?? 'http://localhost:8080'

// One representative write per top-level admin API module. Dummy UUIDs force
// semantic "NotFound" on the backend if wired (distinguishable from gateway
// "Route not found" which is the unwired signal).
//
// `expectedVerdict: 'unwired-mvp'` marks endpoints intentionally scope-cut
// (backend not built; admin-fe toast-rejects client-side per #49/#52).
// Gateway-404 on these is NOT a bug — po-analyst sweep-4 feedback.
const DUMMY_UUID = '00000000-0000-4000-8000-000000000001'
const WRITE_PROBES: WriteProbe[] = [
  // Onboarding: per #47, FE generates UUID client-side and POSTs to
  // /api/onboarding/:id/start with {institutionDetails:{...}}.
  // Bare /api/onboarding is the OLD shape — intentionally gone.
  {
    method: 'POST',
    path: `/api/onboarding/${DUMMY_UUID}/start`,
    body: {
      institutionDetails: {
        name: 'Sweep Probe U',
        slug: 'sweep-probe',
        domain: 'sweep-probe.slate.local',
      },
    },
  },
  { method: 'POST', path: `/api/onboarding/${DUMMY_UUID}/approve`, body: {} },

  // MVP scope-cut: admin-fe toast-rejects these (#49 IAM, #52 admin writes)
  {
    method: 'POST',
    path: '/api/iam/users/invite',
    body: { email: 'probe@example.com', role: 'member' },
    expectedVerdict: 'unwired-mvp',
  },
  {
    method: 'POST',
    path: '/api/iam/roles',
    body: { name: 'sweep-probe-role' },
    expectedVerdict: 'unwired-mvp',
  },

  // Real writes the admin actually uses:
  {
    method: 'POST',
    path: '/api/incidents',
    body: { title: 'sweep probe', severity: 'minor', status: 'open' },
  },
  { method: 'PUT', path: '/api/flags/sweep-probe', body: { enabled: true } },
  {
    method: 'POST',
    path: '/api/broadcast',
    body: { subject: 'probe', body: 'probe', audience: 'admin' },
  },
  {
    method: 'POST',
    path: '/api/tenants',
    body: { slug: 'sweep-probe', name: 'Sweep Probe', domain: 'sweep.slate.local' },
  },

  // Impersonate: R6 registered path is /api/admin/impersonate/:tenant/:user
  // (the /admin/ prefix was missing in earlier probe).
  {
    method: 'POST',
    path: `/api/admin/impersonate/${DUMMY_UUID}/${DUMMY_UUID}`,
    body: { reason: 'sweep' },
  },
]

const ROUTES = [
  // Top nav
  '/ops',
  '/schools',
  '/iam/users',
  '/billing',
  '/incidents',
  '/flags',
  '/iam/audit',
  // Sub-pages
  '/dashboard',
  '/iam/policies',
  '/iam/roles',
  '/impersonation',
  '/broadcast',
  '/status',
  '/onboarding',
  '/onboarding/new',
  '/onboarding/jobs',
  '/onboarding/bulk-import',
  '/onboarding/integrations',
  '/onboard/school',
  '/system/health',
  '/system/alerts',
  '/system/kafka',
  '/data/import',
  '/universities',
  '/admin-service/analytics',
  '/admin-service/billing',
  '/admin-service/plans',
  '/admin-service/optimization',
]

test.describe.configure({ mode: 'serial' })

test('admin portal sweep', async ({ page }) => {
  test.setTimeout(300_000)

  const token = await loginAdmin()
  test.skip(!token, 'admin login failed — stack or creds broken')
  await injectAuth(page, BASE, token!)

  const routes: RouteReport[] = []
  for (const r of ROUTES) {
    const url = BASE + r
    // eslint-disable-next-line no-console
    console.log(`[sweep] admin ${r}`)
    routes.push(await sweepRoute(page, url, 10_000))
  }

  // Write-probe extension is opt-in via SWEEP_WRITES=1 so read-only re-runs
  // stay apples-to-apples with the first-pass inventory.
  let writes: Awaited<ReturnType<typeof runWriteProbes>> | undefined
  if (process.env.SWEEP_WRITES === '1') {
    // eslint-disable-next-line no-console
    console.log(`[sweep] admin running ${WRITE_PROBES.length} write probes`)
    writes = await runWriteProbes(API_URL, token!, WRITE_PROBES)
    for (const w of writes) {
      // eslint-disable-next-line no-console
      console.log(`[sweep] admin write ${w.method} ${w.path} → ${w.status} (${w.verdict})`)
    }
  }

  const report: PortalReport = {
    portal: 'admin',
    baseUrl: BASE,
    loggedIn: true,
    routes,
    writes,
    capturedAt: new Date().toISOString(),
  }
  writeReport('admin', report)
})
