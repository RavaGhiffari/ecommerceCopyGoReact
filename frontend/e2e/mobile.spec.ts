// 390 px (a common phone width): the main buyer pages must not scroll
// sideways. Only a smoke check; the full mobile polish is Phase 3 work.
import type { Page } from '@playwright/test'
import { CABLE, CABLE_NAME } from './support/data'
import { expect, test } from './support/fixtures'

test.use({ viewport: { width: 390, height: 844 } })

const overflow = (page: Page) => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)

test('390 px: home, listing, product, cart and checkout fit the screen', async ({ page, api, customer }) => {
  void customer // signed in, so cart and checkout have content

  for (const [path, ready] of [
    ['/', page.getByRole('heading', { name: 'Produk Terlaris' })],
    ['/products', page.locator('main h3').first()],
    [CABLE, page.getByRole('heading', { name: CABLE_NAME })],
  ] as const) {
    await test.step(`${path} has no horizontal scroll`, async () => {
      await page.goto(path)
      await expect(ready).toBeVisible()
      expect(await overflow(page), `${path} overflows`).toBeLessThanOrEqual(0)
    })
  }

  await api.addAddress()
  const product = await api.get(CABLE)
  await api.data('POST', '/cart/items', { variantId: product.variants[0].id, quantity: 1 })

  await test.step('/cart has no horizontal scroll', async () => {
    await page.goto('/cart')
    await expect(page.getByRole('heading', { name: 'Keranjang', exact: true })).toBeVisible()
    expect(await overflow(page), '/cart overflows').toBeLessThanOrEqual(0)
  })

  await test.step('/checkout has no horizontal scroll', async () => {
    await page.goto('/checkout')
    await expect(page.getByRole('button', { name: 'Buat pesanan' })).toBeVisible()
    expect(await overflow(page), '/checkout overflows').toBeLessThanOrEqual(0)
  })
})
