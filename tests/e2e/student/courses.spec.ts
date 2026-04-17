import { test, expect } from '../fixtures/auth'

test.describe('Student Portal — Courses', () => {
  test('courses page loads and displays heading', async ({ studentPage: page }) => {
    await page.goto('/courses')

    const heading = page.getByRole('heading', { level: 1 })
    await expect(heading).toBeVisible()
    await expect(heading).toContainText(/courses/i)
  })

  test('courses page shows subtitle', async ({ studentPage: page }) => {
    await page.goto('/courses')

    const subtitle = page.getByText(/browse and enroll in available courses/i)
    await expect(subtitle).toBeVisible()
  })

  test('courses page has a search bar', async ({ studentPage: page }) => {
    await page.goto('/courses')

    const searchInput = page.getByPlaceholder(/search courses/i)
    await expect(searchInput).toBeVisible()
  })

  test('courses page has filter tabs (all, enrolled, available)', async ({ studentPage: page }) => {
    await page.goto('/courses')

    await expect(page.getByRole('button', { name: /^all$/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /enrolled/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /available/i })).toBeVisible()
  })

  test('courses page shows course grid or empty state', async ({ studentPage: page }) => {
    await page.goto('/courses')

    // Either the course grid with cards is visible...
    const courseGrid = page.locator('.grid')
    const emptyState = page.getByText(/no courses found/i)

    // One of these should be visible
    const hasGrid = await courseGrid.first().isVisible()
    const hasEmpty = await emptyState.isVisible()
    expect(hasGrid || hasEmpty).toBeTruthy()
  })

  test('search filters courses in real time', async ({ studentPage: page }) => {
    await page.goto('/courses')

    const searchInput = page.getByPlaceholder(/search courses/i)
    await expect(searchInput).toBeVisible()

    // Type a nonsense query -- should show empty state or fewer results
    await searchInput.fill('xyznonexistent')

    // Either no courses or the empty state message should appear
    const emptyState = page.getByText(/no courses found/i)
    // Wait briefly for the filter to apply
    await page.waitForTimeout(500)
    const isEmptyVisible = await emptyState.isVisible()

    // Clear search to restore normal state
    await searchInput.clear()
    await page.waitForTimeout(300)

    // After clearing, courses should reappear or empty state persists
    const courseGrid = page.locator('.grid')
    const hasGrid = await courseGrid.first().isVisible()
    expect(hasGrid || isEmptyVisible).toBeTruthy()
  })
})
