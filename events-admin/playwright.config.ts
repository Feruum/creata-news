import { defineConfig, devices } from '@playwright/test'

const e2ePort = Number(process.env.EVENTS_ADMIN_E2E_PORT || '6868')

export default defineConfig({
  testDir: './e2e',
  testIgnore: ['**/pwa-offline.spec.ts', '**/pwa-coexistence.spec.ts', '**/telemetry-production.spec.ts'],
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',

  use: {
    baseURL: `http://localhost:${e2ePort}`,
    trace: 'on-first-retry'
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'], channel: process.env.PLAYWRIGHT_CHANNEL }
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] }
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] }
    }
  ],

  webServer: {
    command: `node ./node_modules/vite/bin/vite.js --host 127.0.0.1 --port ${e2ePort} --strictPort`,
    url: `http://127.0.0.1:${e2ePort}`,
    reuseExistingServer: !process.env.CI && !process.env.EVENTS_ADMIN_E2E_PORT,
    timeout: 120 * 1000
  }
})
