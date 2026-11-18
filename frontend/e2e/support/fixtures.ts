// Shared test fixtures. Every spec imports `test` and `expect` from here.
//
// - Every page (also those opened with `actor`) is watched for uncaught
//   JavaScript errors; any error fails the test at the end.
// - `actor()` opens another browser (own cookies), e.g. the seller next to the
//   buyer, and closes it after the test.
// - `customer` signs up a fresh customer in the main page's browser, so a
//   spec never depends on what earlier runs left behind.
import { test as base, expect, type Browser, type Page } from '@playwright/test'
import { Api } from './api'
import { SELLER_PASSWORD, uniqueEmail } from './data'

export interface Customer {
  name: string
  email: string
  password: string
  api: Api
}

interface Fixtures {
  pageErrors: string[]
  actor: (options?: { width?: number }) => Promise<Page>
  api: Api
  customer: Customer
}

function watch(page: Page, errors: string[]) {
  page.on('pageerror', (e) => errors.push(`${page.url()}: ${e.message}`))
}

export const test = base.extend<Fixtures>({
  pageErrors: [
    async ({ context }, use) => {
      const errors: string[] = []
      context.pages().forEach((p) => watch(p, errors))
      context.on('page', (p) => watch(p, errors))
      await use(errors)
      expect(errors, 'uncaught errors in the page').toEqual([])
    },
    { auto: true },
  ],

  actor: async ({ browser, pageErrors }, use) => {
    const opened: Awaited<ReturnType<Browser['newContext']>>[] = []
    await use(async ({ width = 1280 } = {}) => {
      const ctx = await browser.newContext({ viewport: { width, height: 1000 } })
      opened.push(ctx)
      const page = await ctx.newPage()
      watch(page, pageErrors)
      return page
    })
    for (const ctx of opened) await ctx.close()
  },

  api: async ({ page }, use) => {
    await use(new Api(page))
  },

  customer: async ({ api }, use) => {
    const c = { name: 'Budi Pembeli', email: uniqueEmail('buyer'), password: 'rahasia-e2e-123', api }
    await api.register(c.name, c.email, c.password)
    await use(c)
  },
})

export { expect }

/** Logs in through the login page and waits until `next` is shown. */
export async function loginUI(page: Page, email: string, password = SELLER_PASSWORD, next = '/') {
  await page.goto('/login?next=' + encodeURIComponent(next))
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Kata sandi').fill(password)
  await page.getByRole('button', { name: 'Masuk' }).click()
  await page.waitForURL((u) => u.pathname === next.split('?')[0])
}

/** Logs out through the account menu and waits for the logged-out header. */
export async function logoutUI(page: Page) {
  await page.locator('header button[aria-haspopup="menu"]').click()
  await page.getByRole('menuitem', { name: 'Keluar' }).click()
  await page.waitForURL((u) => u.pathname === '/')
  await page.getByRole('link', { name: 'Masuk' }).waitFor()
}

/** The header cart link's accessible name, e.g. "Keranjang, 2 barang". */
export const cartBadge = (page: Page) => page.getByRole('link', { name: /^Keranjang/ }).getAttribute('aria-label')
