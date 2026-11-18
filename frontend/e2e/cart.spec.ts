// Phase 4: cart. The seller lowers a demo stock during the test and puts it
// back afterwards (also when the test fails).
import { CABLE, CABLE_NAME, SELLER_BATIK, SELLER_PASSWORD, TSHIRT, TSHIRT_NAME, rupiah, uniqueEmail } from './support/data'
import { apiOf } from './support/api'
import { cartBadge, expect, loginUI, logoutUI, test } from './support/fixtures'

test('cart: login on add, two stores, server totals, stock flags, persistence', async ({ page, actor }) => {
  const email = uniqueEmail('cart')
  const password = 'rahasia-cart-123'
  const summary = page.getByRole('complementary', { name: 'Ringkasan belanja' })

  await test.step('an anonymous add-to-cart asks to log in, and sign-up returns to the product', async () => {
    await page.goto(CABLE)
    await expect(page.getByRole('heading', { name: CABLE_NAME })).toBeVisible()
    await page.getByRole('button', { name: 'Tambah ke keranjang' }).click()
    await page.waitForURL(/\/login\?next=/)
    // The card's link carries ?next; the header's does not.
    await page.getByRole('main').getByRole('link', { name: 'Daftar' }).click()
    await page.getByLabel('Nama').fill('Pembeli Keranjang')
    await page.getByLabel('Email').fill(email)
    await page.getByLabel('Kata sandi').fill(password)
    await page.getByRole('button', { name: 'Daftar' }).click()
    await page.waitForURL((u) => u.pathname === CABLE)
  })

  await test.step('add 2 × "2 meter" cable, then a t-shirt from another store', async () => {
    await page.getByText('2 meter', { exact: true }).click()
    await page.getByRole('button', { name: 'Tambah jumlah' }).click()
    await page.getByRole('button', { name: 'Tambah ke keranjang' }).click()
    await expect(page.getByText('2 × 2 meter ditambahkan ke keranjang.')).toBeVisible()
    await expect.poll(() => cartBadge(page)).toBe('Keranjang, 2 barang')

    await page.goto(TSHIRT)
    await expect(page.getByRole('heading', { name: TSHIRT_NAME })).toBeVisible()
    await page.getByRole('button', { name: 'Tambah ke keranjang' }).click()
    await expect(page.getByText(/ditambahkan ke keranjang/)).toBeVisible()
    await expect.poll(() => cartBadge(page)).toBe('Keranjang, 3 barang')
  })

  await test.step('the cart page groups by store with server totals', async () => {
    await page.getByRole('link', { name: /^Keranjang/ }).click()
    await expect(page.getByRole('heading', { name: 'Keranjang', exact: true })).toBeVisible()
    await expect(page.locator('main section header a')).toHaveCount(2)
    await expect(summary.getByText(rupiah(177000), { exact: true })).toBeVisible()
  })

  await test.step('quantity + and remove update the totals from the server', async () => {
    await page
      .getByRole('group', { name: `Jumlah ${TSHIRT_NAME}` })
      .getByRole('button', { name: 'Tambah jumlah' })
      .click()
    await expect(summary.getByText(rupiah(236000), { exact: true })).toBeVisible()
    await page.getByRole('button', { name: `Hapus ${CABLE_NAME} dari keranjang` }).click()
    await expect(summary.getByText(rupiah(118000), { exact: true })).toBeVisible()
    await expect(page.locator('main section header a')).toHaveCount(1)
  })

  // The seller of the t-shirt lowers the stock of size M below the cart quantity.
  const seller = await actor()
  await loginUI(seller, SELLER_BATIK, SELLER_PASSWORD)
  const sellerApi = apiOf(seller)
  const detail = await sellerApi.get(TSHIRT)
  const variantM = detail.variants.find((v: { name: string }) => v.name === 'M')
  const stockPath = `/seller/products/${detail.id}/variants/${variantM.id}/stock`
  try {
    await test.step('a stock drop below the cart quantity flags the line and leaves it out of the subtotal', async () => {
      await sellerApi.data('PUT', stockPath, { quantity: 1, reason: 'e2e: simulate stock drop' })
      await page.reload()
      await expect(page.getByText('Stok tersisa 1. Kurangi jumlahnya.')).toBeVisible()
      await expect(summary.getByText(rupiah(0), { exact: true })).toBeVisible()
      await page.getByRole('button', { name: 'Ubah ke 1' }).click()
      await expect(summary.getByText(rupiah(59000), { exact: true })).toBeVisible()
      await expect(page.getByText('Stok tersisa 1')).toHaveCount(0)
    })
  } finally {
    await sellerApi.data('PUT', stockPath, { quantity: variantM.stock, reason: 'e2e: restore demo stock' })
  }

  await test.step('a seller cannot add their own product', async () => {
    await seller.goto(TSHIRT)
    await seller.getByRole('button', { name: 'Tambah ke keranjang' }).click()
    await expect(seller.getByText('Ini produk dari tokomu sendiri.')).toBeVisible()
  })

  await test.step('the badge clears on logout and the cart survives a new login', async () => {
    await logoutUI(page)
    await expect.poll(() => cartBadge(page)).toBe('Keranjang')
    await loginUI(page, email, password, '/cart')
    await expect(page.getByRole('heading', { name: 'Keranjang', exact: true })).toBeVisible()
    await expect.poll(() => cartBadge(page)).toBe('Keranjang, 1 barang')
  })
})
