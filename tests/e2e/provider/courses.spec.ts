import { test, expect } from '../fixtures/auth'

test.describe('Provider Portal — Instructor Dashboard & Courses', () => {
  test('instructor dashboard loads with welcome heading', async ({ instructorPage: page }) => {
    await page.goto('/dashboard')

    const heading = page.getByRole('heading', { level: 1 })
    await expect(heading).toBeVisible()
    await expect(heading).toContainText(/welcome back/i)
  })

  test('instructor dashboard shows stat cards', async ({ instructorPage: page }) => {
    await page.goto('/dashboard')

    // The dashboard has stat cards: Active Courses, Total Students, Pending Grades, Avg Class Score
    await expect(page.getByText('Active Courses')).toBeVisible()
    await expect(page.getByText('Total Students')).toBeVisible()
    await expect(page.getByText('Pending Grades')).toBeVisible()
    await expect(page.getByText('Avg Class Score')).toBeVisible()
  })

  test('instructor dashboard shows My Courses section', async ({ instructorPage: page }) => {
    await page.goto('/dashboard')

    const coursesHeading = page.getByRole('heading', { name: /my courses/i })
    await expect(coursesHeading).toBeVisible()

    // "View All" link
    const viewAllLink = page.getByRole('link', { name: /view all/i })
    await expect(viewAllLink).toBeVisible()
  })

  test('courses page loads and shows heading', async ({ instructorPage: page }) => {
    await page.goto('/courses')

    // Heading
    const heading = page.getByRole('heading', { level: 1 })
    await expect(heading).toBeVisible()
    await expect(heading).toContainText(/my courses/i)

    // Subtitle
    const subtitle = page.getByText(/manage and create courses/i)
    await expect(subtitle).toBeVisible()
  })

  test('courses page has Create Course button', async ({ instructorPage: page }) => {
    await page.goto('/courses')

    const createButton = page.getByRole('button', { name: /create course/i })
    await expect(createButton).toBeVisible()
  })

  test('courses page shows course grid or empty state', async ({ instructorPage: page }) => {
    await page.goto('/courses')

    // Either course cards or the "No courses found" empty state
    const courseGrid = page.locator('.grid')
    const emptyState = page.getByText(/no courses found/i)

    const hasGrid = await courseGrid.first().isVisible()
    const hasEmpty = await emptyState.isVisible().catch(() => false)
    expect(hasGrid || hasEmpty).toBeTruthy()
  })

  test('create course form opens when button clicked', async ({ instructorPage: page }) => {
    await page.goto('/courses')

    const createButton = page.getByRole('button', { name: /create course/i })
    await createButton.click()

    // The form should appear with Course Title input
    const titleInput = page.getByLabel(/course title/i)
    await expect(titleInput).toBeVisible()

    // Cancel button to close form
    const cancelButton = page.getByRole('button', { name: /cancel/i })
    await expect(cancelButton).toBeVisible()
  })

  test('sidebar navigation is visible', async ({ instructorPage: page }) => {
    await page.goto('/dashboard')

    // The sidebar has navigation links
    const dashboardLink = page.getByRole('link', { name: /dashboard/i })
    await expect(dashboardLink).toBeVisible()

    const coursesLink = page.getByRole('link', { name: /my courses/i })
    await expect(coursesLink).toBeVisible()

    const studentsLink = page.getByRole('link', { name: /students/i })
    await expect(studentsLink).toBeVisible()

    const gradingLink = page.getByRole('link', { name: /grading/i })
    await expect(gradingLink).toBeVisible()
  })

  test('sidebar has sign out button', async ({ instructorPage: page }) => {
    await page.goto('/dashboard')

    const signOutButton = page.getByRole('button', { name: /sign out/i })
    await expect(signOutButton).toBeVisible()
  })
})
