import { resolve } from 'node:path'
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  testMatch: ['pwa-coexistence.spec.ts', 'telemetry-production.spec.ts'],
  workers: 1,
  timeout: 60_000,
  reporter: 'list',
  use: { baseURL: 'http://127.0.0.1:6872', serviceWorkers: 'allow', trace: 'retain-on-failure' },
  projects: [
    {
      name: 'shared-origin',
      use: {
        ...devices['Desktop Chrome'],
        channel: process.env.PLAYWRIGHT_CHANNEL || (process.platform === 'win32' ? 'chrome' : undefined)
      }
    }
  ],
  webServer: [
    {
      command: 'node e2e/production-server.mjs',
      cwd: resolve(import.meta.dirname, '../map'),
      env: {
        MAP_PWA_SKIP_BUILD: '1',
        EMS_BACKEND_URL: process.env.OTEL_REHEARSAL === '1' ? 'http://127.0.0.1:5556' : 'http://localhost:5555',
        OTEL_EXPORTER_OTLP_ENDPOINT: 'http://127.0.0.1:4318',
        OTEL_COLLECTOR_HTTP_URL: 'http://127.0.0.1:4318'
      },
      url: 'http://127.0.0.1:6870',
      reuseExistingServer: true,
      timeout: 120_000
    },
    {
      command: 'node node_modules/vite/bin/vite.js preview --host 127.0.0.1 --port 6871 --strictPort',
      url: 'http://127.0.0.1:6871/administration/',
      reuseExistingServer: true,
      timeout: 120_000
    },
    {
      command: 'node ../../scripts/pwa-gateway.mjs',
      url: 'http://127.0.0.1:6872/__pwa-test/ready',
      reuseExistingServer: true,
      timeout: 30_000
    }
  ]
})
