import { test as base, type Page } from '@playwright/test'
import {
  ADMIN_CREDENTIALS,
  ensureUser,
  loginUser,
  uniqueEmail,
  type AuthTokens,
} from '../helpers/api'

/**
 * Roles supported by the auth fixture.
 *
 *  - admin:      uses the seeded admin@slate.edu account
 *  - student:    registers a fresh user per test worker
 *  - instructor: registers a fresh user per test worker
 */

export type AuthenticatedFixtures = {
  /** A Page that has been logged in as admin via localStorage token injection */
  adminPage: Page
  /** A Page that has been logged in as a student via localStorage token injection */
  studentPage: Page
  /** A Page that has been logged in as an instructor via localStorage token injection */
  instructorPage: Page
  /** Raw admin tokens for API calls within tests */
  adminTokens: AuthTokens
  /** Raw student tokens for API calls within tests */
  studentTokens: AuthTokens
  /** Raw instructor tokens for API calls within tests */
  instructorTokens: AuthTokens
}

/**
 * Inject an auth token into localStorage before navigating.
 * The frontend apps read `slate_token` for auth state.
 */
async function injectToken(page: Page, baseURL: string, token: string): Promise<void> {
  // Navigate to the base URL first so localStorage is scoped to the correct origin
  await page.goto(baseURL, { waitUntil: 'domcontentloaded' })

  await page.evaluate((tk) => {
    localStorage.setItem('slate_token', tk)
    localStorage.setItem('slate_refresh_token', tk)
  }, token)
}

export const test = base.extend<AuthenticatedFixtures>({
  adminTokens: async ({}, use) => {
    const tokens = await loginUser(ADMIN_CREDENTIALS)
    await use(tokens)
  },

  studentTokens: async ({}, use) => {
    const email = uniqueEmail('student')
    const tokens = await ensureUser({
      email,
      password: 'Student@123456',
      first_name: 'E2E',
      last_name: 'Student',
    })
    await use(tokens)
  },

  instructorTokens: async ({}, use) => {
    const email = uniqueEmail('instructor')
    const tokens = await ensureUser({
      email,
      password: 'Instructor@123456',
      first_name: 'E2E',
      last_name: 'Instructor',
    })
    await use(tokens)
  },

  adminPage: async ({ page, adminTokens }, use) => {
    const baseURL = 'http://localhost:3003'
    await injectToken(page, baseURL, adminTokens.access_token)
    await use(page)
  },

  studentPage: async ({ page, studentTokens }, use) => {
    const baseURL = 'http://localhost:3000'
    await injectToken(page, baseURL, studentTokens.access_token)
    await use(page)
  },

  instructorPage: async ({ page, instructorTokens }, use) => {
    const baseURL = 'http://localhost:3002'
    await injectToken(page, baseURL, instructorTokens.access_token)
    await use(page)
  },
})

export { expect } from '@playwright/test'
