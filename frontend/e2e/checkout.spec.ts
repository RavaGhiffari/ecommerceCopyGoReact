// Phase 5e: checkout -> payment -> orders -> seller fulfilment -> refund / cancel.
// Buyer: a fresh customer whose address is in Surabaya, so only Batik Kita
// (Surabaya) offers Same Day; Gadget Nusantara (Jakarta Barat) does not.
import type { Page } from '@playwright/test'
import { CABLE, SELLER_GADGET, TSHIRT, TSHIRT_NAME, rupiah } from './support/data'
import { cartBadge, expect, loginUI, test } from './support/fixtures'

const storeSection = (page: Page, name: string) =>
  page.locator('main section').filter({ has: page.getByRole('link', { name }) })

test('checkout two stores, pay, fulfil, refund, cancel', async ({ page, api, customer, actor }) => {
  void customer // signed in by the fixture
  const LABEL = `E2E ${Date.now()}`
  const summary = page.getByRole('complementary', { name: 'Ringkasan pesanan' })
  let addressId = ''
  let checkoutId = ''
  const order = { cable: { id: '', number: '' }, tshirt: { id: '', number: '' } }

  await test.step('add to cart, then "Beli sekarang" opens /checkout', async () => {
    await page.goto(CABLE)
    await page.getByRole('button', { name: 'Tambah ke keranjang' }).click()
    await expect(page.getByText(/ditambahkan ke keranjang/)).toBeVisible()
    await page.goto(TSHIRT)
    await expect(page.getByRole('heading', { name: TSHIRT_NAME })).toBeVisible()
    await page.getByRole('button', { name: 'Beli sekarang' }).click()
    await page.waitForURL((u) => u.pathname === '/checkout')
    await expect(page.getByRole('heading', { name: 'Checkout' })).toBeVisible()
  })

  await test.step('an address added from the checkout page gets selected', async () => {
    await page.getByRole('button', { name: 'Alamat baru' }).click()
    const d = page.getByRole('dialog')
    await d.getByLabel('Label').fill(LABEL)
    await d.getByLabel('Nama penerima').fill('Pembeli E2E')
    await d.getByLabel('Nomor telepon').fill('081234567890')
    await d.getByLabel('Alamat lengkap').fill('Jl. Tunjungan No. 1')
    await d.getByLabel('Kota/kabupaten').fill('Surabaya')
    await d.getByLabel('Provinsi').fill('Jawa Timur')
    await d.getByLabel('Kode pos').fill('60261')
    await d.getByRole('button', { name: 'Simpan alamat' }).click()
    await expect(d).toBeHidden()
    await expect(page.getByRole('radio', { name: `Alamat ${LABEL}` })).toHaveAttribute('aria-checked', 'true')
    addressId = (await api.get('/me/addresses')).find((a: { label: string }) => a.label === LABEL)?.id
    expect(addressId).toBeTruthy()
  })

  await test.step('two store groups; Same Day only for the Surabaya store', async () => {
    await expect(storeSection(page, 'Batik Kita').getByRole('radio', { name: /^Same Day/ })).toBeVisible()
    await expect(storeSection(page, 'Gadget Nusantara').getByRole('radio')).toHaveCount(2)
    await expect(storeSection(page, 'Batik Kita').getByRole('radio')).toHaveCount(3)
  })

  await test.step('choosing Same Day re-quotes; the summary equals the server quote', async () => {
    const batikId = (await api.get('/stores/batik-kita')).id
    const sameDay = storeSection(page, 'Batik Kita').getByRole('radio', { name: /^Same Day/ })
    await sameDay.click()
    const quote = await api.data('POST', '/checkout/quote', { addressId, shipping: { [batikId]: 'same_day' } })
    await expect(summary.getByText(rupiah(quote.grandTotal), { exact: true })).toBeVisible()
    await expect(summary.getByText(rupiah(quote.shippingTotal), { exact: true })).toBeVisible()
    await expect(sameDay).toHaveAttribute('aria-checked', 'true')
  })

  await test.step('a double click on "Buat pesanan" creates ONE checkout with 2 orders', async () => {
    await summary.getByRole('button', { name: 'Buat pesanan' }).dblclick()
    await page.waitForURL(/\/checkout\/[0-9a-f-]{36}$/)
    checkoutId = page.url().split('/').pop()!
    await expect(page.getByRole('heading', { name: 'Pembayaran' })).toBeVisible()
    await expect(page.getByText('2 pesanan dibuat. Selesaikan pembayaran.')).toBeVisible()
    await expect(page.getByTestId('checkout-status')).toHaveText('Menunggu pembayaran')
    await expect(page.getByRole('timer', { name: 'Sisa waktu pembayaran' })).toBeVisible()
    expect((await api.get('/orders')).total).toBe(2)
  })

  await test.step('the cart badge is empty after checkout', async () => {
    await expect.poll(() => cartBadge(page)).toBe('Keranjang')
  })

  await test.step('a simulated successful payment -> Lunas, both orders Dibayar', async () => {
    await page.getByRole('button', { name: 'Simulasikan bayar berhasil' }).click()
    await expect(page.getByText('Pembayaran berhasil.', { exact: false }).first()).toBeVisible()
    await expect(page.getByTestId('checkout-status')).toHaveText('Lunas')
    const perStore = page.locator('section[aria-labelledby="orders-heading"]')
    await expect(perStore.getByText('Dibayar', { exact: true })).toHaveCount(2)
    await expect(page.getByRole('button', { name: 'Simulasikan bayar berhasil' })).toHaveCount(0)

    for (const o of (await api.get(`/checkout/${checkoutId}`)).orders) {
      const ref = o.store.slug === 'gadget-nusantara' ? order.cable : order.tshirt
      ref.id = o.id
      ref.number = o.orderNumber
    }
    expect(order.cable.id && order.tshirt.id).toBeTruthy()
  })

  await test.step('"Pesanan saya" in the account menu lists both orders', async () => {
    await page.locator('header button[aria-haspopup="menu"]').click()
    await page.getByRole('menuitem', { name: 'Pesanan saya' }).click()
    await page.waitForURL((u) => u.pathname === '/orders')
    await expect(page.getByRole('article', { name: `Pesanan ${order.cable.number}` })).toBeVisible()
    await expect(page.getByRole('article', { name: `Pesanan ${order.tshirt.number}` })).toBeVisible()
  })

  const seller = await actor()
  await test.step('the seller finds it under "Perlu diproses" and processes it', async () => {
    await loginUI(seller, SELLER_GADGET, undefined, '/seller')
    await seller.getByRole('link', { name: 'Pesanan toko' }).click()
    await seller.waitForURL((u) => u.pathname === '/seller/orders')
    await seller.getByRole('button', { name: 'Perlu diproses' }).click()
    await seller.waitForURL(/status=paid/)
    await seller.getByRole('link', { name: `Detail ${order.cable.number}` }).click()
    await expect(seller.getByRole('heading', { name: order.cable.number })).toBeVisible()
    await seller.getByRole('button', { name: 'Proses pesanan' }).click()
    await expect(seller.getByRole('button', { name: 'Tandai dikirim' })).toBeVisible()
  })

  const resi = `Resi E2E-${Date.now()}`
  await test.step('the seller marks it shipped with a resi note', async () => {
    await seller.getByLabel('Nomor resi / catatan (opsional)').fill(resi)
    await seller.getByRole('button', { name: 'Tandai dikirim' }).click()
    await expect(seller.getByRole('button', { name: 'Tandai selesai' })).toBeVisible()
    await expect(seller.getByRole('list', { name: 'Riwayat status' }).getByText(resi)).toBeVisible()
  })

  await test.step('the buyer sees Dikirim + the resi and can no longer cancel', async () => {
    await page.goto(`/orders/${order.cable.id}`)
    await expect(page.getByRole('heading', { name: order.cable.number })).toBeVisible()
    const timeline = page.getByRole('list', { name: 'Riwayat status' })
    await expect(timeline.getByText(resi)).toBeVisible()
    await expect(timeline.locator('li p:first-child')).toHaveText(['Dikirim', 'Diproses', 'Dibayar', 'Menunggu pembayaran'])
    await expect(page.getByRole('button', { name: 'Batalkan pesanan' })).toHaveCount(0)
  })

  await test.step('the buyer cancels the other (paid) order -> refund', async () => {
    await page.goto(`/orders/${order.tshirt.id}`)
    await page.getByRole('button', { name: 'Batalkan pesanan' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Ya, batalkan' }).click()
    await expect(page.getByText('Pesanan dibatalkan. Dana dikembalikan (simulasi).')).toBeVisible()
    await expect(page.getByRole('main').getByText('Dana dikembalikan', { exact: true }).first()).toBeVisible()
  })

  await test.step("the seller cannot open another store's order (404 from the API)", async () => {
    await seller.goto(`/seller/orders/${order.tshirt.id}`)
    await expect(seller.getByText('ORDER_NOT_FOUND')).toBeVisible()
  })

  await test.step('a new unpaid checkout is cancelled from the payment page', async () => {
    await page.goto(CABLE)
    await page.getByRole('button', { name: 'Beli sekarang' }).click()
    await page.waitForURL((u) => u.pathname === '/checkout')
    await summary.getByRole('button', { name: 'Buat pesanan' }).click()
    await page.waitForURL(/\/checkout\/[0-9a-f-]{36}$/)
    await page.getByRole('button', { name: 'Batalkan checkout' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Ya, batalkan' }).click()
    await expect(page.getByText('Checkout dibatalkan.')).toBeVisible()
    await expect(page.getByTestId('checkout-status')).toHaveText('Dibatalkan')
  })

  await test.step('"Lanjut ke checkout" on the cart page works for a clean cart', async () => {
    await page.goto(CABLE)
    await page.getByRole('button', { name: 'Tambah ke keranjang' }).click()
    await expect(page.getByText(/ditambahkan ke keranjang/)).toBeVisible()
    await page.goto('/cart')
    await page.getByRole('button', { name: 'Lanjut ke checkout' }).click()
    await page.waitForURL((u) => u.pathname === '/checkout')
  })
})
