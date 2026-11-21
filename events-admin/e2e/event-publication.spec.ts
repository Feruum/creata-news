import { expect, type Locator, type Page, test } from '@playwright/test'

// A cold Vite server compiles the full event form before its first fixture load.
test.setTimeout(60_000)

const hosts = [
  { slug: 'campus-club', id: 2, title: 'Campus club', publishOnMap: true, publishOnSundayOpen: false },
  { slug: 'sunday-open', id: 1, title: 'Sunday Open', publishOnMap: false, publishOnSundayOpen: true }
]
const storedTargets = [
  { name: 'map', publishOnMap: true, publishOnSundayOpen: false },
  { name: 'sunday', publishOnMap: false, publishOnSundayOpen: true },
  { name: 'both', publishOnMap: true, publishOnSundayOpen: true }
]

async function expectAutomaticPublication(form: Locator) {
  await expect(form).toBeVisible()
  await expect(form.getByRole('group', { name: 'Publish on', exact: true })).toHaveCount(0)
  await expect(form.getByText('Publish on', { exact: true })).toHaveCount(0)
  await expect(form.getByRole('checkbox', { name: 'map.webspace.cat', exact: true })).toHaveCount(0)
  await expect(form.getByRole('checkbox', { name: 'Sunday Open', exact: true })).toHaveCount(0)
}

async function expectRequest(page: Page, method: string, input: Record<string, unknown>) {
  const requests = JSON.parse(await page.getByTestId('publication-requests').innerText()) as Array<{
    method: string
    input: Record<string, unknown>
  }>
  expect(requests).toHaveLength(1)
  expect(requests[0]).toMatchObject({ method, input })
  expect(requests[0]?.input).not.toHaveProperty('publishOnMap')
  expect(requests[0]?.input).not.toHaveProperty('publishOnSundayOpen')
}

for (const width of [1280, 390]) {
  test.describe(`automatic publication at ${width}px`, () => {
    test.use({ viewport: { width, height: 900 } })

    for (const host of hosts) {
      test(`creating an event for ${host.title} leaves publication to the server`, async ({ page }, testInfo) => {
        await page.goto(`/e2e/fixtures/event-publication.html?host=${host.slug}`)
        const form = page.getByRole('dialog', { name: 'Create Event', exact: true })
        await expectAutomaticPublication(form)
        await expect(form.getByRole('group', { name: 'Primary host', exact: true }).getByRole('combobox')).toHaveText(
          host.title
        )
        const title = `${host.title} workshop`
        await form.getByRole('textbox', { name: 'Title', exact: true }).fill(title)
        await page.screenshot({ path: testInfo.outputPath('automatic-publication-create.png') })
        await form.getByRole('button', { name: 'Create Event', exact: true }).click()
        await expect(form).not.toBeVisible()
        await expectRequest(page, 'create', { title, organizationId: host.id })
        await expect(page.getByTestId('saved-publication')).toHaveText(
          JSON.stringify({ publishOnMap: host.publishOnMap, publishOnSundayOpen: host.publishOnSundayOpen })
        )
      })

      test(`saving a draft for ${host.title} leaves publication to the server`, async ({ page }) => {
        await page.goto(`/e2e/fixtures/event-publication.html?host=${host.slug}`)
        const form = page.getByRole('dialog', { name: 'Create Event', exact: true })
        await expectAutomaticPublication(form)
        const title = `${host.title} draft`
        await form.getByRole('textbox', { name: 'Title', exact: true }).fill(title)
        await form.getByRole('button', { name: 'Save as Draft', exact: true }).click()
        await expect(form).not.toBeVisible()
        await expectRequest(page, 'create', { title, organizationId: host.id, status: 'EVENT_STATUS_DRAFT' })
        await expect(page.getByTestId('saved-publication')).toHaveText(
          JSON.stringify({ publishOnMap: host.publishOnMap, publishOnSundayOpen: host.publishOnSundayOpen })
        )
      })
    }

    for (const targets of storedTargets) {
      test(`editing and reopening a ${targets.name}-platform event preserves stored publication`, async ({
        page
      }, testInfo) => {
        await page.goto(`/e2e/fixtures/event-publication.html?edit=true&targets=${targets.name}`)
        const details = page.getByRole('dialog', { name: 'Sunday workshop', exact: true })
        await details.getByRole('button', { name: 'Edit event', exact: true }).click()
        const form = page.getByRole('dialog', { name: 'Edit Event', exact: true })
        await expectAutomaticPublication(form)
        const title = `Updated ${targets.name} workshop`
        await form.getByRole('textbox', { name: 'Title', exact: true }).fill(title)
        await page.screenshot({ path: testInfo.outputPath('automatic-publication-edit.png') })
        await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
        await expect(form).not.toBeVisible()
        await expectRequest(page, 'update', { id: 9, title })
        await expect(page.getByTestId('saved-publication')).toHaveText(
          JSON.stringify({ publishOnMap: targets.publishOnMap, publishOnSundayOpen: targets.publishOnSundayOpen })
        )
        const savedDetails = page.getByRole('dialog', { name: title, exact: true })
        await savedDetails.getByRole('button', { name: 'Edit event', exact: true }).click()
        await expectAutomaticPublication(form)
        await expect(form.getByRole('textbox', { name: 'Title', exact: true })).toHaveValue(title)
        await expect(page.getByTestId('saved-publication')).toHaveText(
          JSON.stringify({ publishOnMap: targets.publishOnMap, publishOnSundayOpen: targets.publishOnSundayOpen })
        )
      })
    }
  })
}

test.describe('published event changes and form failures', () => {
  test('metadata edits of legacy campus events do not require a new booking', async ({ page }) => {
    await page.goto('/e2e/fixtures/event-publication.html?edit&campus&seconds')
    await page.getByRole('button', { name: 'Edit event', exact: true }).click()
    const form = page.getByRole('dialog', { name: 'Edit Event', exact: true })
    await form.getByRole('textbox', { name: 'Title', exact: true }).fill('Metadata update')
    await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
    await expect(form).not.toBeVisible()
    await expectRequest(page, 'update', {
      title: 'Metadata update',
      startTime: '2027-10-21T12:00:37.654Z',
      endTime: '2027-10-21T13:00:52.456Z',
      structuredLocation: { roomId: 'C1.2.321' }
    })
  })

  for (const failure of ['unavailable', 'permission']) {
    test(`a ${failure} update retains values and displays exactly one owned error`, async ({ page }) => {
      await page.goto(`/e2e/fixtures/event-publication.html?edit&failUpdate=${failure}`)
      await page.getByRole('button', { name: 'Edit event', exact: true }).click()
      const form = page.getByRole('dialog', { name: 'Edit Event', exact: true })
      await form.getByRole('textbox', { name: 'Title', exact: true }).fill('Keep my changes')
      await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
      await expect(form.getByRole('alert')).toHaveCount(1)
      await expect(form.getByRole('alert')).toHaveText(
        failure === 'permission'
          ? 'You do not have permission to perform this action.'
          : 'Could not connect to the server. Check your connection and try again.'
      )
      await expect(form.getByRole('textbox', { name: 'Title', exact: true })).toHaveValue('Keep my changes')
      await expect(page.locator('[data-sonner-toast]')).toHaveCount(0)
      await expect(page.getByText(/database room_booking_id/)).toHaveCount(0)
    })
  }

  test('proposed campus changes wait for approval and save only after a later submission', async ({ page }) => {
    await page.goto('/e2e/fixtures/event-publication.html?edit&slowReadiness')
    await page.getByRole('button', { name: 'Edit event', exact: true }).click()
    const form = page.getByRole('dialog', { name: 'Edit Event', exact: true })
    await form.getByRole('button', { name: 'Campus Room', exact: true }).click()
    await form.getByRole('textbox', { name: 'Room ID', exact: true }).fill('с1.2.321')
    await form.locator('form').evaluate((element: HTMLFormElement) => {
      element.requestSubmit()
      element.requestSubmit()
    })
    await expect(
      form.getByText(
        'An approved room booking is required for this room and time. Request room approval to continue.',
        { exact: true }
      )
    ).toBeVisible()
    let requests = JSON.parse(await page.getByTestId('publication-requests').innerText()) as Array<{
      method: string
      input: Record<string, unknown>
    }>
    expect(requests.map((request) => request.method)).toEqual(['readiness'])
    expect(requests[0]?.input).toMatchObject({
      proposedLocation: { type: 1, roomId: 'C1.2.321' },
      proposedStartTime: expect.any(String),
      proposedEndTime: expect.any(String)
    })
    await form.getByRole('button', { name: 'Request room approval', exact: true }).click()
    await expect(
      form.getByText('Waiting for room approval. Your event has not changed. Save again after approval.', {
        exact: true
      })
    ).toBeVisible()
    requests = JSON.parse(await page.getByTestId('publication-requests').innerText())
    expect(requests.filter((request) => request.method === 'update')).toHaveLength(0)
    expect(requests.filter((request) => request.method === 'booking')).toHaveLength(1)
    await page.evaluate(() => window.dispatchEvent(new Event('approve-booking')))
    await form.getByRole('button', { name: 'Refresh room approval', exact: true }).click()
    await expect(
      form.getByText('Room approval is ready. Save Changes to apply your edits.', { exact: true })
    ).toBeVisible()
    await expect(form).toBeVisible()
    await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
    await expect(form).not.toBeVisible()
    requests = JSON.parse(await page.getByTestId('publication-requests').innerText())
    expect(requests.filter((request) => request.method === 'update')).toHaveLength(1)
  })
})

test.describe('upload validation and owned creation errors', () => {
  test('a failed create retains values and blocks same-tick duplicate requests', async ({ page }) => {
    await page.goto('/e2e/fixtures/event-publication.html?failCreate')
    const form = page.getByRole('dialog', { name: 'Create Event', exact: true })
    await form.getByRole('textbox', { name: 'Title', exact: true }).fill('Keep this new event')
    await form.locator('form').evaluate((element: HTMLFormElement) => {
      element.requestSubmit()
      element.requestSubmit()
    })
    await expect(form.getByRole('alert')).toHaveText(
      'Could not connect to the server. Check your connection and try again.'
    )
    await expect(form.getByRole('textbox', { name: 'Title', exact: true })).toHaveValue('Keep this new event')
    await expect(page.locator('[data-sonner-toast]')).toHaveCount(0)
    const requests = JSON.parse(await page.getByTestId('publication-requests').innerText()) as Array<{ method: string }>
    expect(requests.map((request) => request.method)).toEqual(['create'])
  })

  test('an empty cover image is rejected before upload or create', async ({ page }) => {
    await page.goto('/e2e/fixtures/event-publication.html?emptyImage')
    const form = page.getByRole('dialog', { name: 'Create Event', exact: true })
    await form.getByRole('textbox', { name: 'Title', exact: true }).fill('Event with empty image')
    await form.getByRole('button', { name: 'Create Event', exact: true }).click()
    await expect(form.getByText('Choose a nonempty image', { exact: true })).toBeVisible()
    await expect(page.getByTestId('publication-requests')).toHaveText('[]')
  })

  test('submitting during an image upload creates just one event', async ({ page }) => {
    await page.goto('/e2e/fixtures/event-publication.html?upload&slowUpload')
    const form = page.getByRole('dialog', { name: 'Create Event', exact: true })
    await form.getByRole('textbox', { name: 'Title', exact: true }).fill('Event with cover')
    await form.locator('form').evaluate((element: HTMLFormElement) => {
      element.requestSubmit()
      element.requestSubmit()
    })
    await expect(form).not.toBeVisible()
    const requests = JSON.parse(await page.getByTestId('publication-requests').innerText()) as Array<{ method: string }>
    expect(requests.map((request) => request.method)).toEqual(['upload', 'create'])
  })

  test('an image network failure is inline and retains the form', async ({ page }) => {
    await page.goto('/e2e/fixtures/event-publication.html?upload&failUpload')
    const form = page.getByRole('dialog', { name: 'Create Event', exact: true })
    await form.getByRole('textbox', { name: 'Title', exact: true }).fill('Retry image event')
    await form.getByRole('button', { name: 'Create Event', exact: true }).click()
    await expect(form.getByRole('alert')).toHaveText(
      'Could not connect to the server. Check your connection and try again.'
    )
    await expect(form.getByRole('textbox', { name: 'Title', exact: true })).toHaveValue('Retry image event')
    await expect(page.locator('[data-sonner-toast]')).toHaveCount(0)
    const requests = JSON.parse(await page.getByTestId('publication-requests').innerText()) as Array<{ method: string }>
    expect(requests.map((request) => request.method)).toEqual(['upload'])
  })

  test('denied proposed bookings show one actionable message and preserve the proposal', async ({ page }) => {
    await page.goto('/e2e/fixtures/event-publication.html?edit&denyBooking')
    await page.getByRole('button', { name: 'Edit event', exact: true }).click()
    const form = page.getByRole('dialog', { name: 'Edit Event', exact: true })
    await form.getByRole('button', { name: 'Campus Room', exact: true }).click()
    await form.getByRole('textbox', { name: 'Room ID', exact: true }).fill('C1.2.321')
    await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
    await form.getByRole('button', { name: 'Request room approval', exact: true }).click()
    await expect(form.getByRole('alert')).toHaveText('You do not have permission to perform this action.')
    await expect(form.getByRole('textbox', { name: 'Room ID', exact: true })).toHaveValue('C1.2.321')
    await expect(page.locator('[data-sonner-toast]')).toHaveCount(0)
    const requests = JSON.parse(await page.getByTestId('publication-requests').innerText()) as Array<{ method: string }>
    expect(requests.filter((request) => request.method === 'booking')).toHaveLength(1)
    expect(requests.filter((request) => request.method === 'update')).toHaveLength(0)
  })

  test('format changes drop hidden campus fields from proposed and saved locations', async ({ page }) => {
    await page.goto('/e2e/fixtures/event-publication.html?edit&campus')
    await page.getByRole('button', { name: 'Edit event', exact: true }).click()
    const form = page.getByRole('dialog', { name: 'Edit Event', exact: true })
    await form.getByRole('button', { name: 'Online', exact: true }).click()
    await form.getByRole('textbox', { name: 'Meeting URL', exact: true }).fill('https://example.test/new-meeting')
    await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
    await expect(form).not.toBeVisible()
    const requests = JSON.parse(await page.getByTestId('publication-requests').innerText()) as Array<{
      method: string
      input: { structuredLocation?: Record<string, unknown>; proposedLocation?: Record<string, unknown> }
    }>
    expect(requests.map((request) => request.method)).toEqual(['readiness', 'update'])
    for (const request of requests) {
      const location = request.input.proposedLocation ?? request.input.structuredLocation
      expect(location).not.toHaveProperty('roomId')
      expect(location).not.toHaveProperty('roomName')
      expect(location).not.toHaveProperty('floor')
      expect(location).toHaveProperty('onlineUrl', 'https://example.test/new-meeting')
    }
  })
})

test('approval revoked after preflight refreshes the proposed booking state without losing edits', async ({ page }) => {
  await page.goto('/e2e/fixtures/event-publication.html?edit&autoApprove&revokeOnSave')
  await page.getByRole('button', { name: 'Edit event', exact: true }).click()
  const form = page.getByRole('dialog', { name: 'Edit Event', exact: true })
  await form.getByRole('button', { name: 'Campus Room', exact: true }).click()
  await form.getByRole('textbox', { name: 'Room ID', exact: true }).fill('C1.2.321')
  await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await form.getByRole('button', { name: 'Request room approval', exact: true }).click()
  await expect(
    form.getByText('Room approval is ready. Save Changes to apply your edits.', { exact: true })
  ).toBeVisible()
  await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(
    form.getByText('Waiting for room approval. Your event has not changed. Save again after approval.', { exact: true })
  ).toBeVisible()
  await expect(form.getByRole('textbox', { name: 'Room ID', exact: true })).toHaveValue('C1.2.321')
  await expect(page.locator('[data-sonner-toast]')).toHaveCount(0)
})
test('retrying a created draft booking performs no additional upload, create or update', async ({ page }) => {
  await page.goto('/e2e/fixtures/event-publication.html?campus&upload&failBookingOnce&autoApprove')
  const form = page.getByRole('dialog', { name: 'Create Event', exact: true })
  await form.getByRole('textbox', { name: 'Title', exact: true }).fill('Keep the uploaded campus draft')
  await form.getByRole('button', { name: 'Create Event', exact: true }).click()
  await expect(form.getByRole('alert')).toContainText('Draft saved. Room booking request failed.')
  await expect(form.getByRole('textbox', { name: 'Title', exact: true })).toBeDisabled()
  await expect(form.getByRole('textbox', { name: 'Room ID', exact: true })).toBeDisabled()
  await expect(form.getByRole('group', { name: 'Schedule', exact: true }).getByRole('button').first()).toBeDisabled()
  await expect(form.getByRole('status')).toHaveText(
    'Your draft is saved. Retry room booking to continue, or open the draft to edit it.'
  )
  await expect(form.getByRole('button', { name: 'Retry room booking', exact: true })).toBeEnabled()
  await form.getByRole('button', { name: 'Retry room booking', exact: true }).click()
  await expect(form).not.toBeVisible()
  const requests = JSON.parse(await page.getByTestId('publication-requests').innerText()) as Array<{ method: string }>
  expect(requests.map((request) => request.method)).toEqual([
    'upload',
    'create',
    'booking',
    'booking',
    'readiness',
    'publish'
  ])
})
test('booking review clears after RHF location buttons or room picker changes', async ({ page }) => {
  const selectWarnings: string[] = []
  page.on('console', (message) => {
    if (message.type() === 'error' && /Select.*controlled|uncontrolled.*Select/is.test(message.text())) {
      selectWarnings.push(message.text())
    }
  })
  await page.goto('/e2e/fixtures/event-publication.html?edit')
  await page.getByRole('button', { name: 'Edit event', exact: true }).click()
  const form = page.getByRole('dialog', { name: 'Edit Event', exact: true })
  await form.getByRole('button', { name: 'Campus Room', exact: true }).click()
  await form.getByRole('textbox', { name: 'Room ID', exact: true }).fill('C1.2.321')
  await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(form.getByRole('region', { name: 'Proposed room approval', exact: true })).toBeVisible()
  await form.getByRole('button', { name: 'Online', exact: true }).click()
  await expect(form.getByRole('region', { name: 'Proposed room approval', exact: true })).toHaveCount(0)
  await form.getByRole('button', { name: 'Campus Room', exact: true }).click()
  await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(form.getByRole('region', { name: 'Proposed room approval', exact: true })).toBeVisible()
  await form.getByRole('button', { name: 'Select on Map', exact: true }).click()
  await page
    .getByRole('dialog', { name: 'Select Campus Room', exact: true })
    .getByRole('button', { name: /New review room/ })
    .click()
  await expect(form.getByRole('textbox', { name: 'Room ID', exact: true })).toHaveValue('C1.2.322')
  await expect(form.getByRole('region', { name: 'Proposed room approval', exact: true })).toHaveCount(0)
  await form.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(form.getByRole('region', { name: 'Proposed room approval', exact: true })).toBeVisible()
  const startDate = form.getByRole('group', { name: 'Schedule', exact: true }).getByRole('button').first()
  const initialDate = await startDate.innerText()
  await startDate.click()
  await page
    .locator('[data-slot="calendar"] button[data-day]:not([disabled]):not([data-selected-single="true"])')
    .first()
    .click()
  await page.keyboard.press('Escape')
  await expect(startDate).not.toHaveText(initialDate)
  await expect(form.getByRole('region', { name: 'Proposed room approval', exact: true })).toHaveCount(0)
  const requests = JSON.parse(await page.getByTestId('publication-requests').innerText()) as Array<{ method: string }>
  expect(requests.filter((request) => ['update', 'upload'].includes(request.method))).toHaveLength(0)
  expect(selectWarnings).toEqual([])
})
