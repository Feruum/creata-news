import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  testMatch: 'club-preview*.spec.ts',
  workers: 1,
  retries: 0,
  reporter: 'list',
  globalTeardown: './e2e/club-preview-teardown.mjs',
  outputDir: './test-results/club-preview',
  use: {
    baseURL: 'http://127.0.0.1:6886',
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
  ],
  webServer: {
    command: 'bun ../map/e2e/club-management-server.mjs',
    env: { CLUB_FIXTURE_PORT: '6886' },
    url: 'http://127.0.0.1:6886/admin',
    reuseExistingServer: false,
    timeout: 60_000
  }
})
