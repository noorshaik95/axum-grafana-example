import { test, expect } from '../fixtures/auth'

test.describe('Student Portal — Grades', () => {
  test('grades page loads and displays heading', async ({ studentPage: page }) => {
    await page.goto('/grades')

    const heading = page.getByRole('heading', { level: 1 })
    await expect(heading).toBeVisible()
    await expect(heading).toContainText(/grades/i)
  })

  test('grades page shows subtitle', async ({ studentPage: page }) => {
    await page.goto('/grades')

    const subtitle = page.getByText(/track your academic performance/i)
    await expect(subtitle).toBeVisible()
  })

  test('grades page shows summary stat cards', async ({ studentPage: page }) => {
    await page.goto('/grades')

    // The page has 3 stat cards: GPA, Total Credits, Graded Assignments
    await expect(page.getByText('GPA')).toBeVisible()
    await expect(page.getByText('Total Credits')).toBeVisible()
    await expect(page.getByText('Graded Assignments')).toBeVisible()
  })

  test('grades page shows statistics grid with 3 cards', async ({ studentPage: page }) => {
    await page.goto('/grades')

    // The stat cards are in a 3-column grid
    const statsGrid = page.locator('.grid.gap-4.sm\\:grid-cols-3')
    await expect(statsGrid).toBeVisible()

    const statCards = statsGrid.locator('> div')
    const count = await statCards.count()
    expect(count).toBeGreaterThanOrEqual(3)
  })

  test('grades page shows a grades table', async ({ studentPage: page }) => {
    await page.goto('/grades')

    // The grades are displayed in a table with column headers
    const table = page.locator('table')
    await expect(table).toBeVisible()

    // Check table headers: Course, Assignment, Score, Letter Grade, Date
    await expect(page.getByText('Course')).toBeVisible()
    await expect(page.getByText('Assignment')).toBeVisible()
    await expect(page.getByText('Score')).toBeVisible()
    await expect(page.getByText('Letter Grade')).toBeVisible()
  })

  test('grades table shows empty state or grade rows', async ({ studentPage: page }) => {
    await page.goto('/grades')

    const table = page.locator('table')
    await expect(table).toBeVisible()

    // Either rows with data or the "No grades yet" empty state
    const emptyState = page.getByText(/no grades yet/i)
    const rows = page.locator('tbody tr')
    const rowCount = await rows.count()

    // Should have either the empty state message or data rows
    const hasEmpty = await emptyState.isVisible().catch(() => false)
    expect(rowCount > 0 || hasEmpty).toBeTruthy()
  })
})
