import { EventStatus } from '@studyverse/proto'
import { expect, test } from 'playwright/test'

test.beforeEach(async ({ page }) => {
  await page.route('**/*', (route) => {
    const origin = new URL(route.request().url()).origin
    if (origin !== `http://localhost:${process.env.EVENTS_ADMIN_E2E_PORT || '6868'}`) return route.abort()
    return route.continue()
  })
})

test('invalid draft attempt does not turn the next normal submission into a draft', async ({ page }) => {
  await page.goto('/e2e/fixtures/admin-workflows.html')
  await page.getByLabel('Title', { exact: true }).fill('x')
  await page.getByRole('button', { name: 'Save as Draft', exact: true }).click()
  await expect(page.getByText('Title must be at least 3 characters')).toBeVisible()
  await expect(page.getByTestId('requests')).toHaveText('[]')
  await page.getByLabel('Title', { exact: true }).fill('Valid event title')
  await page.getByRole('button', { name: 'Create Event', exact: true }).click()
  await expect(page.getByTestId('requests')).toHaveText('[{}]')
  await expect(page.getByRole('dialog')).toHaveCount(0)
})

test('failed draft RPC and cancel/reopen preserve the current publish intent', async ({ page }) => {
  await page.goto('/e2e/fixtures/admin-workflows.html?failFirst=true')
  await page.getByLabel('Title', { exact: true }).fill('Changed event title')
  await page.getByRole('button', { name: 'Save as Draft', exact: true }).click()
  await expect(page.getByTestId('requests')).toHaveText(JSON.stringify([{ status: EventStatus.DRAFT }]))
  await expect(page.getByRole('button', { name: 'Cancel', exact: true })).toBeEnabled()
  await page.getByRole('button', { name: 'Cancel', exact: true }).click()
  await page.getByRole('button', { name: 'Open event form', exact: true }).click()
  await page.getByRole('button', { name: 'Create Event', exact: true }).click()
  await expect(page.getByTestId('requests')).toHaveText(JSON.stringify([{ status: EventStatus.DRAFT }, {}]))
  await expect(page.getByRole('dialog')).toHaveCount(0)
})

test('campus creation publishes only after draft creation and approved booking', async ({ page }) => {
  await page.goto('/e2e/fixtures/admin-workflows.html?campus=true')
  await page.getByLabel('Title', { exact: true }).fill('Campus event title')
  await page.getByRole('button', { name: 'Create Event', exact: true }).click()
  await expect(page.getByTestId('workflow')).toHaveText(
    JSON.stringify(['create:draft', 'booking', 'readiness', 'publish'])
  )
  await expect(page.getByText('Event published with an approved room booking!', { exact: true })).toBeVisible()
  await expect(page.getByRole('dialog')).toHaveCount(0)
})

test('pending room approval preserves and opens the created draft', async ({ page }) => {
  await page.goto('/e2e/fixtures/admin-workflows.html?campus=true&pending=true')
  await page.getByLabel('Title', { exact: true }).fill('Pending campus event')
  await page.getByRole('button', { name: 'Create Event', exact: true }).click()
  await expect(page.getByTestId('workflow')).toHaveText(JSON.stringify(['create:draft', 'booking', 'readiness']))
  await expect(page.getByTestId('created-draft')).toHaveText('9')
  await expect(page.getByText('Draft saved. Waiting for room approval.', { exact: true })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Open draft to review or edit' })).toHaveAttribute(
    'href',
    '/events?event=9'
  )
})

test('failed booking retries the saved campus draft without creating another event', async ({ page }) => {
  await page.goto('/e2e/fixtures/admin-workflows.html?campus=true&failBookingOnce=true')
  await page.getByLabel('Title', { exact: true }).fill('Retry campus event')
  await page.getByRole('button', { name: 'Create Event', exact: true }).click()
  await expect(page.getByTestId('created-draft')).toHaveText('9')
  await expect(page.getByRole('alert')).toHaveText(
    'Draft saved. Room booking request failed. Retry to continue with this draft.'
  )
  await page.getByRole('button', { name: 'Retry room booking', exact: true }).click()
  await expect(page.getByTestId('workflow')).toHaveText(
    JSON.stringify(['create:draft', 'booking', 'booking', 'readiness', 'publish'])
  )
  const requests = JSON.parse(await page.getByTestId('requests').innerText()) as Array<{ status?: EventStatus }>
  expect(requests.filter((request) => request.status === EventStatus.DRAFT)).toHaveLength(1)
  expect(requests).toHaveLength(1)
})

test('calendar writes moved and resized dates and replaces them after refetch', async ({ page }) => {
  await page.goto('/e2e/fixtures/admin-workflows.html?calendar=true')
  await page.getByText('Drag event', { exact: true }).dragTo(page.getByText('Drop event here', { exact: true }))
  await expect(page.getByTestId('calendar-dates')).toContainText('2026-10-03T09:00:00.000Z/2026-10-03T10:00:00.000Z')
  await page.getByRole('button', { name: 'Resize event', exact: true }).click()
  await expect(page.getByTestId('calendar-dates')).toContainText('2026-10-03T11:00:00.000Z')
  await expect(page.getByTestId('requests')).toHaveText(
    JSON.stringify([
      { startTime: '2026-10-03T09:00:00.000Z', endTime: '2026-10-03T10:00:00.000Z' },
      { startTime: '2026-10-03T09:00:00.000Z', endTime: '2026-10-03T11:00:00.000Z' }
    ])
  )
  await page.getByRole('button', { name: 'Refetch events', exact: true }).click()
  await expect(page.getByTestId('calendar-dates')).toContainText('2026-10-04T09:00:00.000Z')
})

test('calendar rolls back after a rejected mutation', async ({ page }) => {
  await page.goto('/e2e/fixtures/admin-workflows.html?calendar=true')
  await page.getByRole('button', { name: 'Fail move', exact: true }).click()
  await expect(page.getByTestId('calendar-status')).toHaveText('failed')
  await expect(page.getByTestId('calendar-dates')).toHaveText('2026-10-02T09:00:00.000Z/2026-10-02T10:00:00.000Z')
})

test('resize saves once and keeps handle geometry in sync after rollback and refetch', async ({ page }) => {
  for (const fail of [false, true]) {
    await page.goto(`/e2e/fixtures/admin-workflows.html?calendar=true${fail ? '&rejectUpdates=true' : ''}`)
    const handle = page.getByTestId('resize-event').locator('div[style*="cursor: ns-resize"]').last()
    const bounds = await handle.boundingBox()
    if (!bounds) throw new Error('Resize handle is not visible')
    await page.mouse.move(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2)
    await page.mouse.down()
    await page.mouse.move(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2 + 48, { steps: 4 })
    await expect(page.getByTestId('requests')).toHaveText('[]')
    await page.mouse.up()
    await expect.poll(async () => JSON.parse((await page.getByTestId('requests').textContent()) || '[]').length).toBe(1)
    if (fail) {
      await expect(page.getByTestId('calendar-dates')).toHaveText('2026-10-02T09:00:00.000Z/2026-10-02T10:00:00.000Z')
      await expect(page.getByText('Failed to update event', { exact: true })).toBeVisible()
    } else {
      await expect(page.getByTestId('calendar-dates')).not.toHaveText(
        '2026-10-02T09:00:00.000Z/2026-10-02T10:00:00.000Z'
      )
      await expect(page.getByText('Event updated successfully', { exact: true })).toBeVisible()
    }
    const body = page.getByTestId('resize-body')
    await expect
      .poll(async () => {
        const currentHandle = await handle.boundingBox()
        const currentBody = await body.boundingBox()
        if (!currentHandle || !currentBody) return Infinity
        return Math.abs(currentHandle.y + currentHandle.height / 2 - currentBody.y - currentBody.height)
      })
      .toBeLessThan(1)
    if (fail) await expect(body).toHaveCSS('height', '88px')
    await page.getByRole('button', { name: 'Refetch events', exact: true }).click()
    await expect(body).toHaveCSS('height', '184px')
    await expect
      .poll(async () => {
        const currentHandle = await handle.boundingBox()
        const currentBody = await body.boundingBox()
        if (!currentHandle || !currentBody) return Infinity
        return Math.abs(currentHandle.y + currentHandle.height / 2 - currentBody.y - currentBody.height)
      })
      .toBeLessThan(1)
  }
})
