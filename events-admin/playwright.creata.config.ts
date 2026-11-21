import { defineConfig } from '@playwright/test'
import { resolve } from 'node:path'
import { existsSync } from 'node:fs'

const bundledBrowser = resolve(import.meta.dirname, '../.runtime/browsers/chromium_headless_shell-1208/chrome-headless-shell-win64/chrome-headless-shell.exe')
const executablePath = process.env.CREATA_BROWSER_EXECUTABLE ||
  (!process.env.PLAYWRIGHT_BROWSERS_PATH && process.platform === 'win32' && existsSync(bundledBrowser) ? bundledBrowser : undefined)

export default defineConfig({
  testDir: './e2e-creata', timeout: 60_000, workers: 1, retries: 0, reporter: 'list',
  use: { baseURL: process.env.CREATA_EDITORIAL_TEST_URL || 'http://localhost:6868', viewport: { width: 1440, height: 1000 }, headless: true, launchOptions: { executablePath }, trace: 'retain-on-failure' },
})
