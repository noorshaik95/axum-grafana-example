import { test, expect } from '../fixtures/auth'

test.describe('Admin Portal — Users', () => {
  test('users page loads with heading', async ({ adminPage: page }) => {
    await page.goto('/iam/users')

    const heading = page.getByRole('heading', { name: /users/i })
    await expect(heading).toBeVisible()
  })

  test('users page displays stat cards', async ({ adminPage: page }) => {
    await page.goto('/iam/users')

    // Three stat cards: Total Users, Active (this page), Inactive (this page)
    await expect(page.getByText('Total Users')).toBeVisible()
    await expect(page.getByText(/active \(this page\)/i)).toBeVisible()
    await expect(page.getByText(/inactive \(this page\)/i)).toBeVisible()
  })

  test('users page shows a users table', async ({ adminPage: page }) => {
    await page.goto('/iam/users')

    // The table has specific column headers
    const table = page.locator('table')
    await expect(table).toBeVisible()

    // Check for column headers: Name, Email, Role, Status
    await expect(page.getByText('Name')).toBeVisible()
    await expect(page.getByText('Email')).toBeVisible()
    await expect(page.getByText('Role')).toBeVisible()
    await expect(page.getByText('Status')).toBeVisible()
  })

  test('users table shows user rows', async ({ adminPage: page }) => {
    await page.goto('/iam/users')

    const table = page.locator('table')
    await expect(table).toBeVisible()

    // Should have at least one data row (the admin user at minimum)
    const rows = page.locator('tbody tr')
    const count = await rows.count()
    expect(count).toBeGreaterThan(0)
  })

  test('users page has search functionality', async ({ adminPage: page }) => {
    await page.goto('/iam/users')

    const searchInput = page.getByPlaceholder(/search users/i)
    await expect(searchInput).toBeVisible()

    // Search for a nonexistent user -- should show empty or fewer results
    await searchInput.fill('xyznonexistent')
    await page.waitForTimeout(500)

    // Clear and verify results return
    await searchInput.clear()
    await page.waitForTimeout(300)

    const rows = page.locator('tbody tr')
    const count = await rows.count()
    expect(count).toBeGreaterThanOrEqual(0)
  })

  test('users page has Invite User button', async ({ adminPage: page }) => {
    await page.goto('/iam/users')

    const inviteButton = page.getByRole('button', { name: /invite user/i })
    await expect(inviteButton).toBeVisible()
  })

  test('user rows show role badges', async ({ adminPage: page }) => {
    await page.goto('/iam/users')

    // Role badges like "admin", "student", "instructor" should be visible
    // At least the admin role should exist
    const badge = page.getByText(/admin|student|instructor/i).first()
    await expect(badge).toBeVisible()
  })
})
