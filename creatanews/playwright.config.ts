import { defineConfig,devices } from '@playwright/test'
export default defineConfig({
  testDir:'./e2e',fullyParallel:false,workers:1,timeout:60000,reporter:'list',
  use:{baseURL:process.env.CREATA_WEB_URL||'http://localhost:3002',trace:'retain-on-failure',launchOptions:process.env.CREATA_BROWSER_EXECUTABLE?{executablePath:process.env.CREATA_BROWSER_EXECUTABLE}:undefined},
  projects:[{name:'chromium',use:{...devices['Desktop Chrome'],channel:process.env.PLAYWRIGHT_CHANNEL||(process.platform==='win32'?'chrome':'chromium')}}]
})
