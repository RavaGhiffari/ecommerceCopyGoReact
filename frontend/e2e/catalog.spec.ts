// Phase 2: public catalog on seeded data, then a new seller's journey.
import { uniqueEmail } from './support/data'
import { expect, test } from './support/fixtures'

test('public catalog: best sellers, categories, search, variants', async ({ page, api }) => {
  await test.step('home shows the real best seller first', async () => {
    const top = (await api.get('/products?sort=popular&pageSize=1')).items[0].name
    await page.goto('/')
    await expect(page.locator('section[aria-labelledby="section-popular"] h3').first()).toHaveText(top)
  })

  await test.step('a parent category lists the products of all its subcategories', async () => {
    const parent = (await api.get('/products?category=elektronik&pageSize=1')).total
    const audio = (await api.get('/products?category=audio&pageSize=1')).total
    expect(parent).toBeGreaterThan(audio)

    await page.getByRole('navigation', { name: 'Kategori' }).getByRole('link', { name: 'Elektronik' }).click()
    await page.waitForURL((u) => u.pathname === '/categories/elektronik')
    await expect(page.getByText(`${parent} produk`)).toBeVisible()

    const chips = page.getByRole('navigation', { name: 'Subkategori Elektronik' })
    await chips.getByRole('link', { name: 'Audio' }).click()
    await expect(page.getByText(`${audio} produk`)).toBeVisible()
    await expect(page.locator('main h3')).toContainText(['Earbuds Bluetooth Pro ANC'])
  })

  await test.step('prefix search "kem" finds the batik shirt', async () => {
    await page.getByLabel('Cari produk').fill('kem')
    await page.getByLabel('Cari produk').press('Enter')
    await expect(page.getByRole('heading', { name: 'Hasil pencarian "kem"' })).toBeVisible()
    await expect(page.locator('main h3').filter({ hasText: 'Kemeja Batik Pria Motif Parang' })).toBeVisible()
  })

  await test.step('switching variant shows its own price', async () => {
    await page.getByRole('link', { name: /Kemeja Batik Pria Motif Parang/ }).click()
    await expect(page.getByRole('heading', { name: 'Kemeja Batik Pria Motif Parang' })).toBeVisible()
    await page.getByText('XL', { exact: true }).click()
    await expect(page.getByText('Rp 199.000', { exact: true })).toBeVisible()
  })
})

test('a new seller opens a store, creates, edits and archives a product', async ({ page }) => {
  const stamp = `${Date.now()}`
  const storeName = `Toko Uji ${stamp}`
  const productName = `Headphone Uji ${stamp}`

  await test.step('a new customer at /seller sees "Buka toko"', async () => {
    await page.goto('/register?next=%2Fseller')
    await page.getByLabel('Nama').fill('Penjual Uji')
    await page.getByLabel('Email').fill(uniqueEmail('seller'))
    await page.getByLabel('Kata sandi').fill('rahasia-uji-123')
    await page.getByRole('button', { name: 'Daftar' }).click()
    await expect(page.getByRole('heading', { name: 'Buka toko' })).toBeVisible()
  })

  await test.step('a non-seller is sent from /seller/products/new back to /seller', async () => {
    await page.goto('/seller/products/new')
    await page.waitForURL((u) => u.pathname === '/seller')
  })

  await test.step('opening a store turns the page into the dashboard (role refreshed)', async () => {
    await page.getByLabel('Nama toko').fill(storeName)
    await page.getByLabel('Lokasi').fill('Semarang')
    await page.getByRole('button', { name: 'Buka toko' }).click()
    await expect(page.getByRole('heading', { name: storeName })).toBeVisible()
    await expect(page.getByText('Belum ada produk')).toBeVisible()
    await expect(page.getByRole('link', { name: 'Toko saya' })).toBeVisible()
  })

  await test.step('the browser catches a bad price before the server', async () => {
    await page.getByRole('link', { name: 'Tambah produk' }).click()
    await page.getByLabel('Nama produk').fill(productName)
    await page.getByRole('combobox', { name: 'Kategori' }).click() // Radix Select: open, then pick
    await page.getByRole('option', { name: 'Audio', exact: true }).click()
    await page.getByLabel('Deskripsi').fill('Headphone untuk pengujian end-to-end.')
    await page.getByLabel('Harga (Rp)').first().fill('12,5')
    await page.getByRole('button', { name: 'Simpan produk' }).click()
    await expect(page.getByText('Harga berupa angka', { exact: false }).first()).toBeVisible()
  })

  await test.step('a product with two variants is created and the editor opens', async () => {
    await page.getByLabel('Nama', { exact: true }).first().fill('Hitam')
    await page.getByLabel('SKU').first().fill(`UJI-${stamp}-1`)
    await page.getByLabel('Harga (Rp)').first().fill('125000')
    await page.getByLabel('Stok').first().fill('7')
    await page.getByRole('button', { name: 'Tambah varian' }).click()
    await page.getByLabel('Nama', { exact: true }).nth(1).fill('Putih')
    await page.getByLabel('SKU').nth(1).fill(`UJI-${stamp}-2`)
    await page.getByLabel('Harga (Rp)').nth(1).fill('135000')
    await page.getByLabel('Stok').nth(1).fill('0')
    await page.getByRole('button', { name: 'Simpan produk' }).click()
    await expect(page.getByRole('heading', { name: productName })).toBeVisible()
    await expect(page).toHaveURL(/\/seller\/products\/[0-9a-f-]{36}$/)
  })

  await test.step('stock is set through the editor', async () => {
    const row = page.locator('div.grid', { hasText: 'Hitam' }).filter({ hasText: 'SKU' }).first()
    await row.getByLabel('Stok').fill('3')
    await row.getByRole('button', { name: 'Simpan' }).nth(1).click()
    await expect(page.getByText('stok 3', { exact: false }).first()).toBeVisible()
  })

  await test.step('regression: after a reload the editor shows the saved category', async () => {
    await page.reload()
    await expect(page.getByRole('heading', { name: productName })).toBeVisible()
    await expect(page.getByRole('combobox', { name: 'Kategori' })).toHaveText('Audio')
  })

  await test.step('the product is live in the public catalog', async () => {
    await page.getByRole('link', { name: 'Lihat di katalog' }).click()
    await expect(page.getByRole('heading', { name: productName })).toBeVisible()
    await expect(page.getByText('Rp 125.000', { exact: true })).toBeVisible()
  })

  await test.step('an archived product disappears from public search', async () => {
    await page.goBack()
    await page.getByRole('button', { name: 'Arsipkan' }).click()
    await expect(page.getByText('Diarsipkan', { exact: true })).toBeVisible()
    await page.goto(`/products?q=${encodeURIComponent(`headphone uji ${stamp}`)}`)
    await expect(page.getByText(/Tidak ada hasil/)).toBeVisible()
  })

  await test.step('the server explains a duplicate product name in the same store (409)', async () => {
    await page.goto('/seller/products/new')
    await page.getByLabel('Nama produk').fill(productName)
    await page.getByRole('combobox', { name: 'Kategori' }).click()
    await page.getByRole('option', { name: 'Audio', exact: true }).click()
    await page.getByLabel('Nama', { exact: true }).first().fill('Hitam')
    await page.getByLabel('SKU').first().fill(`UJI-${stamp}-9`)
    await page.getByLabel('Harga (Rp)').first().fill('1000')
    await page.getByRole('button', { name: 'Simpan produk' }).click()
    await expect(page.getByText('Nama ini sudah dipakai', { exact: false })).toBeVisible()
  })
})
