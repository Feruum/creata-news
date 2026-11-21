import { writeFile } from 'node:fs/promises'
import { expect, type Locator, type Page, test } from '@playwright/test'

test.setTimeout(60_000)

async function uploadPoster(page: Page, width = 800, height = 1000) {
  const data = await page.evaluate(
    ({ width, height }) => {
      const canvas = document.createElement('canvas')
      canvas.width = width
      canvas.height = height
      const context = canvas.getContext('2d')!
      context.fillStyle = '#0000ff'
      context.fillRect(0, 0, width, height)
      context.fillStyle = '#ff0000'
      context.fillRect(0, 0, width, 40)
      context.fillStyle = '#00ff00'
      context.fillRect(0, height - 40, width, 40)
      context.fillStyle = '#ffffff'
      context.textAlign = 'center'
      context.font = 'bold 52px sans-serif'
      context.fillText('EVENT POSTER', width / 2, 180)
      context.font = '40px sans-serif'
      context.fillText(`${width} × ${height}`, width / 2, height / 2)
      context.fillText('TOP & BOTTOM VISIBLE', width / 2, height - 130)
      return canvas.toDataURL('image/png').split(',')[1]!
    },
    { width, height }
  )
  const buffer = Buffer.from(data, 'base64')
  await writeFile(test.info().outputPath(`poster-${width}x${height}.png`), buffer)
  await page.locator('input[type=file]').setInputFiles({
    name: 'event-poster.png',
    mimeType: 'image/png',
    buffer
  })
  const editor = page.getByRole('dialog', { name: 'Edit Image', exact: true })
  await expect(editor.locator('.advanced-cropper-background-image')).toBeVisible()
  return editor
}

async function readPreview(page: Page) {
  const preview = page.getByRole('img', { name: 'Upload preview', exact: true })
  await expect(preview).toBeVisible()
  await expect.poll(() => preview.evaluate((image: HTMLImageElement) => image.naturalWidth)).toBeGreaterThan(0)
  return preview.evaluate((image: HTMLImageElement) => {
    const canvas = document.createElement('canvas')
    canvas.width = image.naturalWidth
    canvas.height = image.naturalHeight
    const context = canvas.getContext('2d')!
    context.drawImage(image, 0, 0)
    return {
      width: canvas.width,
      height: canvas.height,
      top: Array.from(context.getImageData(Math.floor(canvas.width / 2), 5, 1, 1).data),
      bottom: Array.from(context.getImageData(Math.floor(canvas.width / 2), canvas.height - 6, 1, 1).data),
      left: Array.from(context.getImageData(5, Math.floor(canvas.height / 2), 1, 1).data),
      right: Array.from(context.getImageData(canvas.width - 6, Math.floor(canvas.height / 2), 1, 1).data)
    }
  })
}

async function expectInViewport(locator: Locator, page: Page) {
  const bounds = await locator.boundingBox()
  expect(bounds).not.toBeNull()
  expect(bounds!.y).toBeGreaterThanOrEqual(0)
  expect(bounds!.y + bounds!.height).toBeLessThanOrEqual(page.viewportSize()!.height)
}

for (const width of [1280, 390]) {
  test.describe(`event poster editor at ${width}px`, () => {
    test.use({
      viewport: { width, height: 664 },
      hasTouch: width === 390,
      isMobile: async ({ browserName }, use) => {
        await use(width === 390 && browserName !== 'firefox')
      }
    })

    test('keeps the entire 4:5 poster including its top and bottom', async ({ page }) => {
      await page.goto('/e2e/fixtures/event-publication.html?host=campus-club')
      const editor = await uploadPoster(page)
      await expect(editor.getByRole('button', { name: '4:5', exact: true })).toHaveAttribute('aria-pressed', 'true')
      await expectInViewport(editor.getByRole('button', { name: 'Apply', exact: true }), page)
      await editor.getByRole('button', { name: 'Apply', exact: true }).click()
      const result = await readPreview(page)
      expect(result.width / result.height).toBeCloseTo(4 / 5, 2)
      expect(result.top[0]).toBeGreaterThan(240)
      expect(result.top[1]).toBeLessThan(15)
      expect(result.bottom[1]).toBeGreaterThan(240)
      expect(result.bottom[0]).toBeLessThan(15)
    })

    test('supports original proportions, landscape crops and cancelling without replacing the image', async ({
      page
    }) => {
      await page.goto('/e2e/fixtures/event-publication.html?host=campus-club')
      let editor = await uploadPoster(page, 600, 1000)
      await editor.getByRole('button', { name: 'Original', exact: true }).click()
      await editor.getByRole('button', { name: 'Apply', exact: true }).click()
      const original = await readPreview(page)
      expect(original.width / original.height).toBeCloseTo(0.6, 2)
      expect(original.top[0]).toBeGreaterThan(240)
      expect(original.bottom[1]).toBeGreaterThan(240)

      editor = await uploadPoster(page)
      await editor.getByRole('button', { name: '16:9', exact: true }).click()
      await editor.getByRole('button', { name: 'Cancel', exact: true }).click()
      expect(await readPreview(page)).toEqual(original)

      editor = await uploadPoster(page)
      await editor.getByRole('button', { name: '16:9', exact: true }).click()
      await editor.getByRole('button', { name: 'Apply', exact: true }).click()
      const landscape = await readPreview(page)
      expect(landscape.width / landscape.height).toBeCloseTo(16 / 9, 2)
    })

    test('fits the full poster after zooming and preserves original proportions when rotated', async ({ page }) => {
      await page.goto('/e2e/fixtures/event-publication.html?host=campus-club')
      let editor = await uploadPoster(page)
      const zoom = editor.getByRole('slider', { name: 'Image zoom' })
      await zoom.press('End')
      await expect(zoom).toHaveAttribute('aria-valuenow', '3')
      await editor.getByRole('button', { name: '4:5', exact: true }).click()
      await expect(zoom).toHaveAttribute('aria-valuenow', '3')
      await editor.getByRole('button', { name: 'Fit image', exact: true }).click()
      await expect(zoom).toHaveAttribute('aria-valuenow', '1')
      await editor.getByRole('button', { name: 'Apply', exact: true }).click()
      const fitted = await readPreview(page)
      expect(fitted.top[0]).toBeGreaterThan(240)
      expect(fitted.bottom[1]).toBeGreaterThan(240)

      editor = await uploadPoster(page, 600, 1000)
      await editor.getByRole('button', { name: 'Original', exact: true }).click()
      await editor.getByRole('button', { name: 'Rotate 90°', exact: true }).click()
      await editor.getByRole('button', { name: '16:9', exact: true }).click()
      await editor.getByRole('button', { name: 'Original', exact: true }).click()
      await editor.getByRole('button', { name: 'Fit image', exact: true }).click()
      await editor.getByRole('button', { name: 'Apply', exact: true }).click()
      const rotated = await readPreview(page)
      expect(rotated.width / rotated.height).toBeCloseTo(1000 / 600, 2)
      expect(rotated.left[1]).toBeGreaterThan(240)
      expect(rotated.right[0]).toBeGreaterThan(240)
    })
  })
}
