import { defineConfig, devices } from '@playwright/test'

// Browser E2E suite (Phase 7a). Runs against its OWN stack: the api-e2e
// container (host port 8082) on the marketplace_e2e database, never the dev
// API or dev data. `make e2e` resets and seeds that database first.
//
// Vite is started here on 127.0.0.1:5191 and proxies /api to api-e2e
// (API_TARGET, see vite.config.ts), so the browser still sees one origin and
// the session cookie and CSRF Origin check work exactly as in development.

const PORT = 5191
export const BASE_URL = `http://127.0.0.1:${PORT}`
export const E2E_API = process.env.E2E_API_URL ?? 'http://127.0.0.1:8082'
const CI = !!process.env.CI

export default defineConfig({
  testDir: './e2e',
  // The specs share one seeded database and a few demo accounts (e.g. the
  // admin spec suspends penjual4, the cart spec lowers a demo stock), so they
  // run one at a time. The whole suite takes a few minutes.
  workers: 1,
  fullyParallel: false,
  forbidOnly: CI,
  retries: CI ? 1 : 0,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  reporter: [['list'], ['html', { open: 'never' }]],
  globalSetup: './e2e/support/global-setup.ts',
  use: {
    baseURL: BASE_URL,
    locale: 'id-ID',
    timezoneId: 'Asia/Jakarta',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1280, height: 1000 },
        // Locally: the installed Microsoft Edge (no browser download).
        // CI: Playwright's own Chromium (`npx playwright install chromium`).
        channel: CI ? undefined : 'msedge',
      },
    },
  ],
  webServer: {
    command: `npx vite --host 127.0.0.1 --port ${PORT} --strictPort`,
    url: BASE_URL,
    // Never reuse whatever happens to listen on 5191: it could proxy to the
    // dev API and the suite would write into the dev database.
    reuseExistingServer: false,
    env: { API_TARGET: E2E_API },
    timeout: 60_000,
  },
})
