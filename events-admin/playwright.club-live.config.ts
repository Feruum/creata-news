import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  testMatch: 'club-live-preview.live.ts',
  workers: 1,
  retries: 0,
  timeout: 180_000,
  reporter: 'list',
  outputDir: './test-results/club-live-preview',
  use: {
    baseURL: 'http://localhost:5173',
    serviceWorkers: 'block',
    actionTimeout: 30_000,
    navigationTimeout: 60_000,
    timezoneId: 'Asia/Almaty',
    trace: 'retain-on-failure'
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        channel: process.env.PLAYWRIGHT_CHANNEL || (process.platform === 'win32' ? 'msedge' : undefined)
      }
    }
  ]
})
