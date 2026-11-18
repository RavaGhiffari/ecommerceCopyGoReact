// Wishlist (practice feature): heart button, /wishlist page, empty state.
import { CABLE, CABLE_NAME, TSHIRT, TSHIRT_NAME, uniqueEmail } from './support/data'
import { expect, test } from './support/fixtures'

test('wishlist: save, list, remove', async ({ page, api }) => {
  // The product itself; "Produk terkait" cards below it have hearts too.
  const product = page.getByRole('article')
  const heart = product.getByRole('button', { name: /favorit$/ })
  const names = page.locator('main li a.line-clamp-2')

  await test.step('signed out, the heart sends you to sign up and back', async () => {
    await page.goto(CABLE)
    await heart.click()
    await page.waitForURL((u) => u.pathname === '/login' && u.searchParams.get('next') === CABLE)
    // A fresh account through the API, then back to the product.
    await api.register('Budi Pembeli', uniqueEmail('wish'), 'rahasia-e2e-123')
    await page.goto(CABLE)
  })

  await test.step('the heart is off; a click saves it (toast, aria-pressed)', async () => {
    await expect(product.getByRole('button', { name: 'Simpan ke favorit' })).toBeVisible()
    await expect(heart).toHaveAttribute('aria-pressed', 'false')
    await heart.click()
    await expect(page.getByText('Disimpan ke favorit.')).toBeVisible()
    await expect(product.getByRole('button', { name: 'Hapus dari favorit' })).toBeVisible()
    await expect(heart).toHaveAttribute('aria-pressed', 'true')
  })

  await test.step('still saved after a reload (comes from the server)', async () => {
    await page.reload()
    await expect(product.getByRole('button', { name: 'Hapus dari favorit' })).toBeVisible()
  })

  await test.step('a second product, then "Favorit saya" from the account menu', async () => {
    await page.goto(TSHIRT)
    await product.getByRole('button', { name: 'Simpan ke favorit' }).click()
    await expect(page.getByText('Disimpan ke favorit.')).toBeVisible()
    await page.locator('header button[aria-haspopup="menu"]').click()
    await page.getByRole('menuitem', { name: 'Favorit saya' }).click()
    await page.waitForURL((u) => u.pathname === '/wishlist')
  })

  await test.step('/wishlist lists both, newest first, with store and price', async () => {
    await expect(page.getByRole('heading', { name: 'Favorit saya' })).toBeVisible()
    await expect(page.getByText('2 produk')).toBeVisible()
    await expect(names).toHaveText([TSHIRT_NAME, CABLE_NAME])
    await expect(page.getByRole('link', { name: 'Gadget Nusantara' })).toBeVisible()
    await expect(page.getByText('Rp 39.000')).toBeVisible()
  })

  await test.step('remove from the list (toast, one left)', async () => {
    await page.getByRole('button', { name: `Hapus ${TSHIRT_NAME} dari favorit` }).click()
    await expect(page.getByText(`${TSHIRT_NAME} dihapus dari favorit.`)).toBeVisible()
    await expect(page.getByText('1 produk')).toBeVisible()
    await expect(names).toHaveCount(1)
  })

  await test.step('the removed product shows an empty heart again', async () => {
    await page.goto(TSHIRT)
    await expect(product.getByRole('button', { name: 'Simpan ke favorit' })).toBeVisible()
  })

  await test.step('unsave from the product page -> empty state on /wishlist', async () => {
    await page.goto(CABLE)
    await product.getByRole('button', { name: 'Hapus dari favorit' }).click()
    await expect(page.getByText('Dihapus dari favorit.')).toBeVisible()
    await page.goto('/wishlist')
    await expect(page.getByRole('heading', { name: 'Belum ada favorit' })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Jelajahi produk' })).toBeVisible()
  })
})
