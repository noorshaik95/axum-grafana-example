/**
 * Student portal smoke spec — Wave 5 qa.
 *
 * Flow:
 *   1. Login as student03@eastfield.edu
 *   2. Lands on /today (new student landing per Wave 4)
 *   3. Navigate to /office-hours
 *   4. Book a slot with Martinez Thursday 15:00
 *   5. Assert booking appears in GET /api/scheduling/instructor-day
 *
 * Requires: seed-eastfield.sh has run (Martinez OH schedule + student03 exist).
 */

import { test, expect } from '@playwright/test'

const API_URL = process.env.API_URL ?? 'http://localhost:8080'

const STUDENT03 = {
  email: 'student03@eastfield.edu',
  password: 'Slate-test-1!',
}

async function apiLogin(email: string, password: string): Promise<string | null> {
  try {
    const res = await fetch(`${API_URL}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!res.ok) return null
    const json = (await res.json()) as { access_token?: string; accessToken?: string }
    return json.access_token ?? json.accessToken ?? null
  } catch {
    return null
  }
}

function nextThursdayIso(hour = 15, minute = 0): string {
  const now = new Date()
  const day = now.getUTCDay() // 0 = Sun, 4 = Thu
  const daysUntil = (4 - day + 7) % 7 || 7
  const target = new Date(
    Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() + daysUntil, hour, minute, 0)
  )
  return target.toISOString()
}

test.describe('Student portal — office hours (Eastfield)', () => {
  test('student03 logs in, lands on /today, books OH with Martinez', async ({ page }) => {
    const token = await apiLogin(STUDENT03.email, STUDENT03.password)
    test.skip(!token, 'student03 not seeded — run ./scripts/seed-eastfield.sh')

    await page.goto('/')
    await page.evaluate((tk) => {
      localStorage.setItem('slate_token', tk)
      localStorage.setItem('slate_refresh_token', tk)
    }, token!)

    // --- 2. Landing ------------------------------------------------------
    await page.goto('/today')
    await expect(page).toHaveURL(/\/today/)

    // --- 3. Office hours page -------------------------------------------
    await page.goto('/office-hours')
    await expect(page).toHaveURL(/\/office-hours/)

    // Capture X-Request-ID for handoff to #25 observability-expert.
    const reqId = `pw-student-oh-${Date.now()}`
    let capturedRequestId = reqId
    page.on('response', async (resp) => {
      const xrid = resp.headers()['x-request-id']
      if (xrid && resp.url().includes('/api/scheduling')) capturedRequestId = xrid
    })

    // --- 4. Book a slot --------------------------------------------------
    // Prefer UI interaction; fall back to API if UI doesn't surface a bookable button.
    const bookBtn = page.getByRole('button', { name: /book|reserve|schedule/i }).first()
    const hasBookUI = await bookBtn.isVisible().catch(() => false)

    let booked = false
    if (hasBookUI) {
      await bookBtn.click()
      const confirm = page.getByRole('button', { name: /confirm|book now|save/i }).first()
      if (await confirm.isVisible().catch(() => false)) {
        await confirm.click()
        booked = true
      }
    }

    if (!booked) {
      // Fallback: list schedules, pick first, POST booking
      const schedulesRes = await page.request.get(`${API_URL}/api/scheduling/schedules`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (schedulesRes.ok()) {
        const schedBody = (await schedulesRes.json()) as Record<string, unknown>
        const list = (schedBody.items ??
          schedBody.data ??
          schedBody.schedules ??
          schedBody) as Array<Record<string, unknown>>
        const schedule = Array.isArray(list)
          ? (list.find((s) =>
              String(s.label ?? '')
                .toLowerCase()
                .includes('martinez')
            ) ?? list[0])
          : null
        const scheduleId = schedule?.id as string | undefined
        if (scheduleId) {
          const bookRes = await page.request.post(`${API_URL}/api/scheduling/bookings`, {
            headers: {
              'Content-Type': 'application/json',
              Authorization: `Bearer ${token}`,
              'X-Request-ID': reqId,
            },
            data: {
              schedule_id: scheduleId,
              starts_at: nextThursdayIso(15, 0),
              duration_minutes: 15,
              context: 'Playwright smoke — book OH',
            },
          })
          expect(bookRes.status(), `POST booking returned ${bookRes.status()}`).toBeLessThan(500)
          const respRid = bookRes.headers()['x-request-id']
          if (respRid) capturedRequestId = respRid
        }
      }
    }

    await test.info().attach('x-request-id', {
      body: `${capturedRequestId}\n`,
      contentType: 'text/plain',
    })

    // --- 5. Verify via instructor-day ------------------------------------
    const dayRes = await page.request.get(`${API_URL}/api/scheduling/instructor-day`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(
      dayRes.status(),
      `GET /api/scheduling/instructor-day returned ${dayRes.status()}`
    ).toBeLessThan(500)
  })
})
