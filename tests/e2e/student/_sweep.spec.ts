/**
 * Student portal sweep (task #40).
 */
import { test } from '@playwright/test'
import {
  injectAuth,
  loginRegular,
  runWriteProbes,
  sweepRoute,
  writeReport,
  type PortalReport,
  type RouteReport,
  type WriteProbe,
} from '../sweep/_sweep-helper'

const BASE = 'http://localhost:3000'
const API_URL = process.env.API_URL ?? 'http://localhost:8080'

// One representative write per top-level student API module.
const DUMMY_UUID = '00000000-0000-4000-8000-000000000003'
const WRITE_PROBES: WriteProbe[] = [
  {
    method: 'POST',
    path: `/api/assignments/${DUMMY_UUID}/submissions`,
    body: { content: 'sweep probe', language: 'python' },
  },
  { method: 'POST', path: `/api/courses/${DUMMY_UUID}/enroll`, body: {} },
  {
    method: 'POST',
    path: '/api/discussions/threads',
    body: { course_id: DUMMY_UUID, title: 'sweep', body: 'probe' },
  },
  {
    method: 'POST',
    path: `/api/discussions/threads/${DUMMY_UUID}/posts`,
    body: { body: 'sweep reply' },
  },
  { method: 'PATCH', path: `/api/discussions/inbox/${DUMMY_UUID}/read`, body: {} },
  {
    method: 'POST',
    path: '/api/scheduling/bookings',
    body: { schedule_id: DUMMY_UUID, starts_at: '2030-01-01T14:00:00Z', duration_minutes: 15 },
  },
  { method: 'POST', path: '/api/ai/study-plan', body: { goal: 'probe', weekly_hours: 4 } },
]

const ROUTES = [
  // Top nav
  '/today',
  '/courses',
  '/assignments',
  '/grades',
  '/plan',
  '/inbox',
  '/people',
  // Secondary
  '/dashboard',
  '/calendar',
  '/discussions',
  '/messages',
  '/announcements',
  '/achievements',
  '/files',
  '/video',
  '/office-hours',
  '/study-groups',
  '/calculator',
  '/profile',
  '/settings',
]

test.describe.configure({ mode: 'serial' })

test('student portal sweep', async ({ page }) => {
  test.setTimeout(300_000)
  // Prefer seeded student account (added by #44) so student middleware
  // accepts us. Fall back to admin if seed hasn't landed.
  const token =
    (await loginRegular('student@slate.edu', 'Student@123456')) ??
    (await loginRegular('admin@slate.edu', 'Admin@123456'))
  test.skip(!token, 'no login worked — stack or creds broken')
  await injectAuth(page, BASE, token!)

  const routes: RouteReport[] = []
  for (const r of ROUTES) {
    // eslint-disable-next-line no-console
    console.log(`[sweep] student ${r}`)
    routes.push(await sweepRoute(page, BASE + r, 10_000))
  }

  let writes: Awaited<ReturnType<typeof runWriteProbes>> | undefined
  if (process.env.SWEEP_WRITES === '1') {
    // eslint-disable-next-line no-console
    console.log(`[sweep] student running ${WRITE_PROBES.length} write probes`)
    writes = await runWriteProbes(API_URL, token!, WRITE_PROBES)
    for (const w of writes) {
      // eslint-disable-next-line no-console
      console.log(`[sweep] student write ${w.method} ${w.path} → ${w.status} (${w.verdict})`)
    }
  }

  writeReport('student', {
    portal: 'student',
    baseUrl: BASE,
    loggedIn: true,
    routes,
    writes,
    capturedAt: new Date().toISOString(),
  })
})
