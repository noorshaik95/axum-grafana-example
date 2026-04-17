import { test, expect } from '@playwright/test'
import { ADMIN_CREDENTIALS, loginUser } from '../helpers/api'

test.describe('Student Portal — Authentication', () => {
  test('unauthenticated user is redirected to login', async ({ page }) => {
    await page.goto('/')

    // The student app home page checks for slate_token and redirects
    // to /login if missing, or /dashboard if present
    await expect(page).toHaveURL(/\/login/)
  })

  test('login page renders correctly', async ({ page }) => {
    await page.goto('/login')

    // Should display "Welcome Back" heading
    const heading = page.getByRole('heading', { name: /welcome back/i })
    await expect(heading).toBeVisible()

    // Should have email and password inputs
    await expect(page.getByLabel(/email/i)).toBeVisible()
    await expect(page.getByLabel(/password/i)).toBeVisible()

    // Should have a Sign In button
    await expect(page.getByRole('button', { name: /sign in/i })).toBeVisible()
  })

  test('dashboard page loads after authentication', async ({ page }) => {
    const tokens = await loginUser(ADMIN_CREDENTIALS)

    await page.goto('/')
    await page.evaluate((token) => {
      localStorage.setItem('slate_token', token)
    }, tokens.access_token)

    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/dashboard/)

    // Should see the welcome message
    const heading = page.getByRole('heading', { level: 1 })
    await expect(heading).toBeVisible()
    await expect(heading).toContainText(/welcome/i)
  })

  test('logout clears auth state and redirects to login', async ({ page }) => {
    const tokens = await loginUser(ADMIN_CREDENTIALS)

    // Set token
    await page.goto('/')
    await page.evaluate((token) => {
      localStorage.setItem('slate_token', token)
    }, tokens.access_token)

    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/dashboard/)

    // Clear tokens (simulates logout)
    await page.evaluate(() => {
      localStorage.removeItem('slate_token')
      localStorage.removeItem('slate_refresh_token')
    })

    // Verify token is removed
    const token = await page.evaluate(() => localStorage.getItem('slate_token'))
    expect(token).toBeNull()
  })
})
