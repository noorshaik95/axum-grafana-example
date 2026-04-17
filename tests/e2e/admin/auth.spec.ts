import { test, expect } from '@playwright/test'
import { ADMIN_CREDENTIALS, loginUser } from '../helpers/api'

test.describe('Admin Portal — Authentication', () => {
  test('login page renders correctly', async ({ page }) => {
    await page.goto('/login')

    // Should display the Admin Dashboard title
    const title = page.getByText('Admin Dashboard')
    await expect(title).toBeVisible()

    // Should have email and password inputs
    const emailInput = page.getByLabel(/email/i)
    await expect(emailInput).toBeVisible()

    const passwordInput = page.getByLabel(/password/i)
    await expect(passwordInput).toBeVisible()

    // Should have a Sign In button
    const submitButton = page.getByRole('button', { name: /sign in/i })
    await expect(submitButton).toBeVisible()
  })

  test('login with valid admin credentials redirects to /dashboard', async ({ page }) => {
    await page.goto('/login')

    // Fill in admin credentials
    await page.getByLabel(/email/i).fill(ADMIN_CREDENTIALS.email)
    await page.getByLabel(/password/i).fill(ADMIN_CREDENTIALS.password)

    // Submit the form
    await page.getByRole('button', { name: /sign in/i }).click()

    // Should redirect to /dashboard on success
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 10000 })
  })

  test('login with invalid credentials shows error', async ({ page }) => {
    await page.goto('/login')

    await page.getByLabel(/email/i).fill('wrong@example.com')
    await page.getByLabel(/password/i).fill('wrongpassword')
    await page.getByRole('button', { name: /sign in/i }).click()

    // Should show an error toast: "Login Failed" or "Invalid credentials"
    const errorToast = page.getByText(/login failed|invalid credentials|access denied/i).first()
    await expect(errorToast).toBeVisible({ timeout: 10000 })
  })

  test('dashboard has admin-specific elements', async ({ page }) => {
    // Login via API and inject token
    const tokens = await loginUser(ADMIN_CREDENTIALS)

    await page.goto('/')
    await page.evaluate((token) => {
      localStorage.setItem('slate_token', token)
      localStorage.setItem('slate_refresh_token', token)
    }, tokens.access_token)

    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/dashboard/)

    // Admin dashboard heading is "Dashboard"
    const heading = page.getByRole('heading', { level: 1 })
    await expect(heading).toBeVisible()
    await expect(heading).toContainText(/dashboard/i)

    // Should show admin stat cards
    await expect(page.getByText('Total Users')).toBeVisible()
    await expect(page.getByText('System Status')).toBeVisible()
  })

  test('logout button is present in sidebar', async ({ page }) => {
    const tokens = await loginUser(ADMIN_CREDENTIALS)

    await page.goto('/')
    await page.evaluate((token) => {
      localStorage.setItem('slate_token', token)
      localStorage.setItem('slate_refresh_token', token)
    }, tokens.access_token)

    // Navigate to a page that has the sidebar
    await page.goto('/iam/users')

    // The sidebar has a Logout button
    const logoutButton = page.getByRole('button', { name: /logout/i })
    await expect(logoutButton).toBeVisible()
  })
})
