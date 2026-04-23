/**
 * Provider portal sweep (task #40). Enumerates top-nav + sub-page routes with a
 * logged-in instructor and captures integration state for po-analyst aggregation.
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

const BASE = 'http://localhost:3002'
const API_URL = process.env.API_URL ?? 'http://localhost:8080'

// One representative write per top-level provider API module. Dummy UUIDs
// force semantic NotFound on the backend when the route IS wired.
const DUMMY_UUID = '00000000-0000-4000-8000-000000000002'
const WRITE_PROBES: WriteProbe[] = [
  {
    method: 'POST',
    path: '/api/courses',
    body: { code: 'SWEEP 101', title: 'Sweep Probe', capacity: 10 },
  },
  {
    method: 'POST',
    path: '/api/assignments',
    body: { course_id: DUMMY_UUID, title: 'Sweep probe', points_possible: 10, state: 'draft' },
  },
  {
    method: 'POST',
    path: '/api/grades',
    body: { assignment_id: DUMMY_UUID, student_id: DUMMY_UUID, score: 50 },
  },
  { method: 'POST', path: `/api/grades/${DUMMY_UUID}/publish`, body: {} },
  {
    method: 'POST',
    path: '/api/scheduling/schedules',
    body: { label: 'sweep OH', slot_minutes: 15, location_kind: 'online', weekly_slots: [] },
  },
  {
    method: 'POST',
    path: '/api/discussions/threads',
    body: { course_id: DUMMY_UUID, title: 'sweep thread', body: 'probe' },
  },
  {
    method: 'POST',
    path: '/api/broadcast',
    body: { subject: 'probe', body: 'probe', audience: 'course' },
  },
  {
    method: 'POST',
    path: '/api/content/upload/initiate',
    body: { filename: 'probe.mp4', content_type: 'video/mp4', size_bytes: 1024 },
  },
]

const ROUTES = [
  // Top nav
  '/teach',
  '/courses',
  '/grade',
  '/roster',
  '/office-hours',
  '/discussion',
  // Secondary
  '/grading',
  '/grading/rules',
  '/analytics',
  '/calendar',
  '/communications',
  '/announcements',
  '/messages',
  '/exports',
  '/video',
  '/lecture/live',
  '/students',
  '/settings',
]

test.describe.configure({ mode: 'serial' })

test('provider portal sweep', async ({ page }) => {
  test.setTimeout(300_000)
  // Prefer seeded instructor (added by #44) so provider middleware's
  // instructor-role gate accepts us. Fall back to admin if seed hasn't landed —
  // the sweep still captures route HTTP + layout presence either way.
  const token =
    (await loginRegular('instructor@slate.edu', 'Instructor@123456')) ??
    (await loginRegular('admin@slate.edu', 'Admin@123456'))
  test.skip(!token, 'no login worked — stack or creds broken')
  await injectAuth(page, BASE, token!)

  const routes: RouteReport[] = []
  for (const r of ROUTES) {
    // eslint-disable-next-line no-console
    console.log(`[sweep] provider ${r}`)
    routes.push(await sweepRoute(page, BASE + r, 10_000))
  }

  let writes: Awaited<ReturnType<typeof runWriteProbes>> | undefined
  if (process.env.SWEEP_WRITES === '1') {
    // eslint-disable-next-line no-console
    console.log(`[sweep] provider running ${WRITE_PROBES.length} write probes`)
    writes = await runWriteProbes(API_URL, token!, WRITE_PROBES)
    for (const w of writes) {
      // eslint-disable-next-line no-console
      console.log(`[sweep] provider write ${w.method} ${w.path} → ${w.status} (${w.verdict})`)
    }
  }

  writeReport('provider', {
    portal: 'provider',
    baseUrl: BASE,
    loggedIn: true,
    routes,
    writes,
    capturedAt: new Date().toISOString(),
  })
})
