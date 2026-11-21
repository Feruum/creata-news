import { expect, type Page, test } from '@playwright/test'

type ReadinessState = 'missing' | 'pending' | 'rejected' | 'cancelled' | 'ready' | 'unavailable'
const draft = {
  id: 901,
  title: 'Local campus draft',
  description: JSON.stringify([{ type: 'p', children: [{ text: 'Synthetic local event description.' }] }]),
  userId: 1,
  organizationId: 1,
  organization: { id: 1, title: 'Local host', slug: 'local-host' },
  startTime: '2026-10-14T20:33:00.000Z',
  endTime: '2026-10-14T21:33:00.000Z',
  createdAt: '2026-10-07T20:33:00.000Z',
  updatedAt: '2026-10-07T20:33:00.000Z',
  format: 'EVENT_FORMAT_OFFLINE',
  status: 'EVENT_STATUS_DRAFT',
  structuredLocation: { type: 'LOCATION_TYPE_CAMPUS_ROOM', roomId: '334L', roomName: '334L', floor: 3 },
  slug: 'local-campus-draft'
}
const e2eOrigin = `http://localhost:${process.env.EVENTS_ADMIN_E2E_PORT || '6868'}`
const messages = {
  missing: 'cannot publish: campus room booking request has not been submitted yet',
  pending: 'cannot publish: room booking request is still pending approval',
  rejected: 'cannot publish: room booking request was rejected',
  cancelled: 'cannot publish: room booking request was cancelled',
  ready: 'room booking approved',
  unavailable: 'Event readiness service is unavailable. Try again.',
  bookingUnavailable: 'Room booking service is unavailable. Try again.',
  publishFailure: 'cannot publish: room booking request was cancelled',
  bookingPermissionDenied: 'No permission to book this room.',
  existingBooking: 'A room booking request already exists for this event.'
}
const readinessStates = {
  missing: 'EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_MISSING',
  pending: 'EVENT_PUBLISH_READINESS_STATE_WAITING_ROOM_APPROVAL',
  rejected: 'EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_REJECTED',
  cancelled: 'EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_CANCELLED',
  ready: 'EVENT_PUBLISH_READINESS_STATE_READY'
}

const displayed = {
  missing: 'An approved room booking is required for this room and time. Request room approval to continue.',
  pending: 'Waiting for room approval before publishing.',
  rejected: 'The room booking was rejected. You can submit a new request or choose another room or time.',
  cancelled: 'The room booking was cancelled. Submit a new request before continuing.',
  ready: 'This draft is ready to publish.',
  unavailable: 'Could not connect to the server. Check your connection and try again.',
  bookingUnavailable: 'Could not connect to the server. Check your connection and try again.',
  publishFailure: 'Room approval is required for this room and time. Check the booking status and try again.'
}

async function mockDraft(
  page: Page,
  options: {
    readiness: ReadinessState
    sharedState?: { readiness: ReadinessState }
    bookingResponseError?: 'unavailable' | 'already_exists'
    bookingPermissionDenied?: boolean
    autoApproved?: boolean
    holdBooking?: boolean
    bookingFails?: boolean
    publishFails?: boolean
    hybrid?: boolean
  }
) {
  let event = {
    ...draft,
    structuredLocation: {
      ...draft.structuredLocation,
      type: options.hybrid ? 'LOCATION_TYPE_HYBRID' : draft.structuredLocation.type
    }
  }
  const serverState = options.sharedState ?? { readiness: options.readiness }
  let bookingFails = options.bookingFails
  let holdReadiness = false
  const workflow: string[] = []
  const publishRequests: unknown[] = []
  const bookingRequests: unknown[] = []
  let releaseReadiness = () => {}
  let releaseBooking = () => {}
  const readinessGate = new Promise<void>((resolve) => {
    releaseReadiness = resolve
  })
  const bookingGate = new Promise<void>((resolve) => {
    releaseBooking = resolve
  })
  await page.route('**/*', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = url.pathname.split('/').at(-1) ?? ''
    const headers = {
      'access-control-allow-origin': e2eOrigin,
      'access-control-allow-credentials': 'true'
    }
    if (url.pathname.endsWith('/sessions/whoami')) {
      return route.fulfill({
        headers,
        json: {
          id: 'local-session',
          active: true,
          identity: { id: 'local-identity', traits: { email: 'qa@example.test', name: 'Local QA' } }
        }
      })
    }
    if (method === 'GetEventPublishReadiness') {
      workflow.push('readiness')
      if (holdReadiness) {
        holdReadiness = false
        await readinessGate
      }
      if (serverState.readiness === 'unavailable') {
        return route.fulfill({ status: 503, headers, json: { code: 'unavailable', message: messages.unavailable } })
      }
      return route.fulfill({
        headers,
        json: {
          canPublishNow: serverState.readiness === 'ready',
          requiresRoomApproval: true,
          state: readinessStates[serverState.readiness],
          message: messages[serverState.readiness],
          ...(serverState.readiness === 'rejected' ? { rejectionReason: 'Room is reserved for an exam.' } : {})
        }
      })
    }
    if (method === 'CreateBookingRequest') {
      workflow.push('booking')
      bookingRequests.push(request.postDataJSON())
      if (options.holdBooking) await bookingGate
      if (options.bookingPermissionDenied) {
        return route.fulfill({
          status: 403,
          headers,
          json: { code: 'permission_denied', message: messages.bookingPermissionDenied }
        })
      }
      if (bookingFails) {
        return route.fulfill({
          status: 503,
          headers,
          json: { code: 'unavailable', message: messages.bookingUnavailable }
        })
      }
      serverState.readiness = options.autoApproved ? 'ready' : 'pending'
      if (options.bookingResponseError) {
        return route.fulfill({
          status: options.bookingResponseError === 'already_exists' ? 409 : 503,
          headers,
          json: {
            code: options.bookingResponseError,
            message:
              options.bookingResponseError === 'already_exists' ? messages.existingBooking : messages.bookingUnavailable
          }
        })
      }
      return route.fulfill({
        headers,
        json: {
          bookingRequest: { id: 999, roomId: draft.structuredLocation.roomId, title: draft.title },
          autoApproved: options.autoApproved ?? false
        }
      })
    }
    if (method === 'PublishEvent') {
      workflow.push('publish')
      publishRequests.push(request.postDataJSON())
      if (options.publishFails) {
        return route.fulfill({
          status: 400,
          headers,
          json: { code: 'failed_precondition', message: messages.publishFailure }
        })
      }
      event = { ...event, status: 'EVENT_STATUS_PUBLISHED' }
      return route.fulfill({ headers, json: { event } })
    }
    const payloads: Record<string, unknown> = {
      GetMe: {
        user: { id: 1, username: 'local-qa', email: 'qa@example.test' },
        permissions: { isAdmin: true, managedClubIds: [1] }
      },
      GlobalSearch: { results: [], totalHits: 0 },
      ListEventsForAdmin: { events: [event], total: 1 },
      GetEvent: { event, canEdit: true, canDelete: true, canManageRegistrations: false },
      ListOrganizations: { organizations: [draft.organization], total: 1 },
      ListCategories: { categories: [], total: 0 },
      ListRooms: { rooms: [], total: 0 }
    }
    if (method in payloads) return route.fulfill({ headers, json: payloads[method] })
    if (url.origin === e2eOrigin) return route.continue()
    return route.abort()
  })
  return {
    workflow,
    publishRequests,
    bookingRequests,
    releaseReadiness,
    holdNextReadiness: () => {
      holdReadiness = true
    },
    releaseBooking,
    setReadiness: (state: ReadinessState) => {
      serverState.readiness = state
    },
    allowBooking: () => {
      bookingFails = false
    }
  }
}

for (const width of [1280, 390]) {
  test.describe(`draft publication at ${width}px`, () => {
    test.beforeEach(async ({ page }) => {
      await page.setViewportSize({ width, height: 900 })
    })

    for (const autoApproved of [false, true]) {
      test(`requests a missing room booking and follows backend ${autoApproved ? 'approval' : 'pending'} state`, async ({
        page
      }, testInfo) => {
        const api = await mockDraft(page, {
          readiness: 'missing',
          autoApproved,
          holdBooking: true,
          hybrid: width === 390 && autoApproved
        })
        await page.goto('/events?event=901')
        const dialog = page.getByRole('dialog')
        const requestBooking = dialog.getByRole('button', { name: 'Request room booking', exact: true })
        const publish = dialog.getByRole('button', { name: width === 390 ? 'Publish' : 'Publish event', exact: true })
        await expect(requestBooking).toBeVisible()
        await expect(dialog.getByText(displayed.missing, { exact: true })).toBeVisible()
        await expect(publish).toBeDisabled()
        await requestBooking.scrollIntoViewIfNeeded()
        await page.screenshot({ path: testInfo.outputPath('missing-room-booking.png') })
        await requestBooking.click()
        await expect.poll(() => api.bookingRequests.length).toBe(1)
        await expect(requestBooking).toBeDisabled()
        await expect(publish).toBeDisabled()
        api.releaseBooking()
        await expect(
          dialog.getByText(autoApproved ? displayed.ready : displayed.pending, { exact: true })
        ).toBeVisible()
        await expect(requestBooking).toHaveCount(0)
        expect(api.bookingRequests).toEqual([
          {
            roomId: draft.structuredLocation.roomId,
            organizationId: draft.organizationId,
            title: draft.title,
            startTime: draft.startTime,
            endTime: draft.endTime,
            eventId: draft.id
          }
        ])
        expect(api.publishRequests).toEqual([])
        await page.screenshot({ path: testInfo.outputPath('room-booking-result.png') })
        if (!autoApproved) {
          await expect(publish).toBeDisabled()
          await dialog.getByRole('button', { name: 'Refresh room approval', exact: true }).click()
          await expect(dialog.getByText(displayed.pending, { exact: true })).toBeVisible()
          await expect(publish).toBeDisabled()
          expect(api.bookingRequests).toHaveLength(1)
          api.setReadiness('ready')
          await dialog.getByRole('button', { name: 'Refresh room approval', exact: true }).click()
          await expect(dialog.getByText(displayed.ready, { exact: true })).toBeVisible()
        }
        await expect(publish).toBeEnabled()
        await publish.click()
        await expect(page.getByText('Event published', { exact: true })).toBeVisible()
        expect(api.publishRequests).toEqual([{ eventId: draft.id }])
        expect(api.bookingRequests).toHaveLength(1)
        await expect(dialog.getByRole('button', { name: 'Publish event', exact: true })).toHaveCount(0)
        await testInfo.attach('workflow', {
          body: JSON.stringify(
            { workflow: api.workflow, bookingRequests: api.bookingRequests, publishRequests: api.publishRequests },
            null,
            2
          ),
          contentType: 'application/json'
        })
        await page.screenshot({ path: testInfo.outputPath('saved-draft-published.png') })
      })
    }

    test('shows pending room approval and cannot create duplicate requests', async ({ page }) => {
      const api = await mockDraft(page, { readiness: 'pending' })
      await page.goto('/events?event=901')
      const dialog = page.getByRole('dialog')
      await expect(dialog.getByText(displayed.pending, { exact: true })).toBeVisible()
      await expect(dialog.getByRole('button', { name: 'Request room booking', exact: true })).toHaveCount(0)
      await expect(dialog.getByRole('button', { name: 'Publish event', exact: true })).toBeDisabled()
      await dialog.getByRole('button', { name: 'Refresh room approval', exact: true }).click()
      await expect(dialog.getByText(displayed.pending, { exact: true })).toBeVisible()
      expect(api.bookingRequests).toEqual([])
      expect(api.publishRequests).toEqual([])
    })

    test('shows room rejection and allows a new request', async ({ page }) => {
      const api = await mockDraft(page, { readiness: 'rejected' })
      await page.goto('/events?event=901')
      const dialog = page.getByRole('dialog')
      await expect(dialog.getByText(displayed.rejected, { exact: true })).toBeVisible()
      await expect(dialog.getByText('Room is reserved for an exam.', { exact: true })).toBeVisible()
      await expect(dialog.getByRole('button', { name: 'Request room booking', exact: true })).toBeEnabled()
      await expect(dialog.getByRole('button', { name: 'Publish event', exact: true })).toBeDisabled()
      expect(api.bookingRequests).toEqual([])
      expect(api.publishRequests).toEqual([])
      await dialog.getByRole('button', { name: 'Request room booking', exact: true }).click()
      await expect(dialog.getByText(displayed.pending, { exact: true })).toBeVisible()
      expect(api.bookingRequests).toHaveLength(1)
    })

    test('allows a new request after a room booking was cancelled', async ({ page }) => {
      const api = await mockDraft(page, { readiness: 'cancelled' })
      await page.goto('/events?event=901')
      const dialog = page.getByRole('dialog')
      await expect(dialog.getByText(displayed.cancelled, { exact: true })).toBeVisible()
      await dialog.getByRole('button', { name: 'Request room booking', exact: true }).click()
      await expect(dialog.getByText(displayed.pending, { exact: true })).toBeVisible()
      expect(api.bookingRequests).toHaveLength(1)
    })

    test('rechecks approval before Publish and blocks publication if approval changed', async ({ page }) => {
      const api = await mockDraft(page, { readiness: 'ready' })
      await page.goto('/events?event=901')
      const dialog = page.getByRole('dialog')
      const headerPublish = dialog.getByRole('button', { name: 'Publish event', exact: true })
      const footerPublish = dialog.getByRole('button', { name: 'Publish', exact: true })
      await expect(headerPublish).toBeEnabled()
      const callsBeforeCheck = api.workflow.length
      api.holdNextReadiness()
      await (width === 390 ? footerPublish : headerPublish).click()
      await expect.poll(() => api.workflow.length).toBeGreaterThan(callsBeforeCheck)
      await expect(headerPublish).toBeDisabled()
      await expect(footerPublish).toBeDisabled()
      expect(api.publishRequests).toEqual([])
      api.setReadiness('pending')
      api.releaseReadiness()
      await expect(dialog.getByText(displayed.pending, { exact: true })).toBeVisible()
      await expect(headerPublish).toBeDisabled()
      expect(api.publishRequests).toEqual([])
      api.setReadiness('ready')
      await dialog.getByRole('button', { name: 'Refresh room approval', exact: true }).click()
      await expect(headerPublish).toBeEnabled()
      await headerPublish.click()
      await expect(page.getByText('Event published', { exact: true })).toBeVisible()
      expect(api.publishRequests).toEqual([{ eventId: draft.id }])
    })

    test('shows a readiness service error and retries after recovery', async ({ page }) => {
      const api = await mockDraft(page, { readiness: 'unavailable' })
      await page.goto('/events?event=901')
      const dialog = page.getByRole('dialog')
      await expect(dialog.getByRole('alert')).toContainText(displayed.unavailable)
      await expect(dialog.getByRole('button', { name: 'Publish event', exact: true })).toBeDisabled()
      expect(api.publishRequests).toEqual([])
      api.setReadiness('ready')
      await dialog.getByRole('button', { name: 'Retry room approval', exact: true }).click()
      await expect(dialog.getByText(displayed.ready, { exact: true })).toBeVisible()
      const publish = dialog.getByRole('button', { name: 'Publish event', exact: true })
      await expect(publish).toBeEnabled()
      api.setReadiness('unavailable')
      await publish.click()
      await expect(dialog.getByRole('alert')).toContainText(displayed.unavailable)
      await expect(publish).toBeDisabled()
      expect(api.publishRequests).toEqual([])
      await expect(page.getByText('Failed to publish event', { exact: true })).toHaveCount(0)
    })

    test('keeps a failed booking request recoverable without changing event dates', async ({ page }) => {
      const api = await mockDraft(page, { readiness: 'missing', bookingFails: true })
      await page.goto('/events?event=901')
      const requestBooking = page.getByRole('dialog').getByRole('button', { name: 'Request room booking', exact: true })
      await requestBooking.click()
      await expect(page.getByText(displayed.bookingUnavailable, { exact: true })).toBeVisible()
      await expect(requestBooking).toBeEnabled()
      expect(api.bookingRequests).toHaveLength(1)
      expect(api.publishRequests).toEqual([])
      api.allowBooking()
      await requestBooking.click()
      await expect(page.getByRole('dialog').getByText(displayed.pending, { exact: true })).toBeVisible()
      expect(api.bookingRequests).toHaveLength(2)
      expect(api.bookingRequests[1]).toEqual(api.bookingRequests[0])
      expect(api.publishRequests).toEqual([])
    })

    test('a stale second tab refreshes room readiness before creating another request', async ({ page, context }) => {
      test.setTimeout(120_000)
      const sharedState: { readiness: ReadinessState } = { readiness: 'missing' }
      const first = await mockDraft(page, { readiness: 'missing', sharedState })
      const secondPage = await context.newPage()
      await secondPage.setViewportSize({ width, height: 900 })
      const second = await mockDraft(secondPage, { readiness: 'missing', sharedState })
      await page.bringToFront()
      await page.goto('/events?event=901')
      const firstRequest = page.getByRole('dialog').getByRole('button', { name: 'Request room booking', exact: true })
      await expect(firstRequest).toBeEnabled()
      await secondPage.bringToFront()
      await secondPage.goto('/events?event=901')
      const secondRequest = secondPage
        .getByRole('dialog')
        .getByRole('button', { name: 'Request room booking', exact: true })
      await expect(secondRequest).toBeEnabled()
      await page.bringToFront()
      await firstRequest.click()
      await expect(page.getByRole('dialog').getByText(displayed.pending, { exact: true })).toBeVisible()
      await secondPage.bringToFront()
      await secondRequest.click()
      await expect(secondPage.getByRole('dialog').getByText(displayed.pending, { exact: true })).toBeVisible()
      await expect(secondRequest).toHaveCount(0)
      expect(first.bookingRequests).toHaveLength(1)
      expect(second.bookingRequests).toEqual([])
      expect(second.publishRequests).toEqual([])
      await secondPage.close()
    })

    for (const bookingResponseError of ['unavailable', 'already_exists'] as const) {
      test(`refreshes committed room readiness after a ${bookingResponseError} booking response`, async ({ page }) => {
        const api = await mockDraft(page, { readiness: 'missing', bookingResponseError })
        await page.goto('/events?event=901')
        const dialog = page.getByRole('dialog')
        const requestBooking = dialog.getByRole('button', { name: 'Request room booking', exact: true })
        await requestBooking.click()
        if (bookingResponseError === 'unavailable') {
          await expect(page.getByRole('alert')).toHaveText(displayed.bookingUnavailable)
        } else {
          await expect(page.getByRole('alert')).toHaveCount(0)
        }
        await expect(dialog.getByText(displayed.pending, { exact: true })).toBeVisible()
        await expect(requestBooking).toHaveCount(0)
        await expect(dialog.getByRole('button', { name: 'Publish event', exact: true })).toBeDisabled()
        await dialog.getByRole('button', { name: 'Refresh room approval', exact: true }).click()
        await expect(dialog.getByText(displayed.pending, { exact: true })).toBeVisible()
        expect(api.bookingRequests).toHaveLength(1)
        expect(api.publishRequests).toEqual([])
      })
    }

    test('a room booking permission error uses one owned inline message', async ({ page }) => {
      const api = await mockDraft(page, { readiness: 'missing', bookingPermissionDenied: true })
      await page.goto('/events?event=901')
      const requestBooking = page.getByRole('dialog').getByRole('button', { name: 'Request room booking', exact: true })
      await requestBooking.click()
      await expect(page.getByText('You do not have permission to perform this action.', { exact: true })).toBeVisible()
      await expect(requestBooking).toBeEnabled()
      await expect(page.locator('[data-sonner-toast]')).toHaveCount(0)
      await expect(page.getByRole('alert')).toHaveCount(1)
      await expect(page.getByText(messages.bookingPermissionDenied, { exact: true })).toHaveCount(0)
      expect(api.bookingRequests).toHaveLength(1)
      expect(api.publishRequests).toEqual([])
    })

    test('preserves an actual Publish rejection without a duplicate generic toast', async ({ page }) => {
      const api = await mockDraft(page, { readiness: 'ready', publishFails: true })
      await page.goto('/events?event=901')
      const publish = page.getByRole('dialog').getByRole('button', { name: 'Publish event', exact: true })
      await expect(publish).toBeEnabled()
      const callsBeforePublish = api.workflow.length
      await publish.click()
      await expect(page.getByRole('alert')).toHaveText(displayed.publishFailure)
      await expect(publish).toBeEnabled()
      await expect(page.getByText('Failed to publish event', { exact: true })).toHaveCount(0)
      expect(api.workflow.slice(callsBeforePublish)).toEqual(['readiness', 'publish', 'readiness'])
      expect(api.publishRequests).toEqual([{ eventId: draft.id }])
      expect(api.bookingRequests).toEqual([])
      await expect(page.locator('[data-sonner-toast]')).toHaveCount(0)

      api.setReadiness('pending')
      const callsBeforeRetry = api.workflow.length
      await publish.click()
      await expect(page.getByRole('dialog').getByText(displayed.pending, { exact: true })).toBeVisible()
      await expect(publish).toBeDisabled()
      expect(api.workflow.slice(callsBeforeRetry)).toEqual(['readiness'])
      expect(api.publishRequests).toEqual([{ eventId: draft.id }])
      expect(api.bookingRequests).toEqual([])
    })
  })
}
