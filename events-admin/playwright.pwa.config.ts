import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  testMatch: 'pwa-offline.spec.ts',
  fullyParallel: false,
  workers: 1,
  timeout: 45_000,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:6871/administration/',
    serviceWorkers: 'allow',
    trace: 'retain-on-failure'
  },
  projects: [
    {
      name: 'pwa-chromium',
      use: {
        ...devices['Desktop Chrome'],
        channel: process.env.PLAYWRIGHT_CHANNEL || (process.platform === 'win32' ? 'chrome' : undefined)
      }
    }
  ],
  webServer: {
    env: { VITE_OTEL_ENABLED: 'true' },
    // Exercises the compiled production bundle and real injectManifest service worker.
    command: `${process.env.PWA_SKIP_BUILD ? '' : 'node node_modules/vite/bin/vite.js build && '}node node_modules/vite/bin/vite.js preview --host 127.0.0.1 --port 6871 --strictPort`,
    url: 'http://127.0.0.1:6871/administration/',
    reuseExistingServer: !!process.env.PWA_SKIP_BUILD,
    timeout: 180_000
  }
})
