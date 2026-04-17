import { test, expect } from '../fixtures/auth'

test.describe('Student Portal — Dashboard', () => {
  test('dashboard loads and displays welcome message', async ({ studentPage: page }) => {
    await page.goto('/dashboard')

    // The dashboard has a welcome heading "Welcome back, <name>!"
    const heading = page.getByRole('heading', { level: 1 })
    await expect(heading).toBeVisible()
    await expect(heading).toContainText(/welcome/i)
  })

  test('dashboard displays stat cards', async ({ studentPage: page }) => {
    await page.goto('/dashboard')

    // The dashboard has 4 stat cards: Enrolled Courses, Assignments Due, GPA, Study Streak
    await expect(page.getByText('Enrolled Courses')).toBeVisible()
    await expect(page.getByText('Assignments Due')).toBeVisible()
    await expect(page.getByText('GPA')).toBeVisible()
    await expect(page.getByText('Study Streak')).toBeVisible()
  })

  test('dashboard shows My Courses section', async ({ studentPage: page }) => {
    await page.goto('/dashboard')

    // The "My Courses" section heading
    const coursesHeading = page.getByRole('heading', { name: /my courses/i })
    await expect(coursesHeading).toBeVisible()

    // "View all" link to /courses
    const viewAllLink = page.getByRole('link', { name: /view all/i })
    await expect(viewAllLink).toBeVisible()
  })

  test('dashboard shows Upcoming section', async ({ studentPage: page }) => {
    await page.goto('/dashboard')

    const upcomingHeading = page.getByRole('heading', { name: /upcoming/i })
    await expect(upcomingHeading).toBeVisible()
  })

  test('dashboard renders stat cards in a grid', async ({ studentPage: page }) => {
    await page.goto('/dashboard')

    // The stat cards are in a 4-column grid
    const statGrid = page.locator('.grid.gap-4').first()
    await expect(statGrid).toBeVisible()

    const gridItems = statGrid.locator('> div')
    const count = await gridItems.count()
    expect(count).toBeGreaterThanOrEqual(4)
  })

  test('sidebar navigation is visible', async ({ studentPage: page }) => {
    await page.goto('/dashboard')

    // The sidebar has navigation links for Dashboard, Courses, etc.
    const dashboardLink = page.getByRole('link', { name: /dashboard/i })
    await expect(dashboardLink).toBeVisible()

    const coursesLink = page.getByRole('link', { name: /courses/i })
    await expect(coursesLink).toBeVisible()

    const gradesLink = page.getByRole('link', { name: /grades/i })
    await expect(gradesLink).toBeVisible()
  })

  test('dashboard has quick action links', async ({ studentPage: page }) => {
    await page.goto('/dashboard')

    // Quick action cards at the bottom: Browse Courses, View Assignments, Check Grades, Study Groups
    await expect(page.getByText('Browse Courses')).toBeVisible()
    await expect(page.getByText('View Assignments')).toBeVisible()
    await expect(page.getByText('Check Grades')).toBeVisible()
  })
})
