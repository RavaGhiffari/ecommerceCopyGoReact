// Phase 6b: write a review from /account, see it on the product page, edit it
// from the order page; rating sort; no review button before delivery.
// Setup goes through the API: a fresh buyer buys the cable, the seller
// (penjual1, Gadget Nusantara) delivers it; a second order stays "paid".
import { CABLE, SELLER_GADGET, TSHIRT } from './support/data'
import { apiOf } from './support/api'
import { expect, loginUI, test } from './support/fixtures'

test('reviews: write, show, edit, sort, rules', async ({ page, api, customer, actor }) => {
  void customer // "Budi Pembeli", masked as "Budi P***"
  const stamp = Date.now()
  const COMMENT = `Ulasan E2E ${stamp}: kabel kokoh.`
  const EDITED = `Ulasan E2E ${stamp}: diubah jadi 2 bintang.`

  const seller = await actor()
  await loginUI(seller, SELLER_GADGET)
  const address = await api.addAddress()
  const delivered = (await api.buyAndPay(CABLE, address.id)).orders[0]
  await apiOf(seller).advanceOrder(delivered.id, 'delivered')
  const undelivered = (await api.buyAndPay(TSHIRT, address.id)).orders[0]
  const item = delivered.items[0]
  const before = (await api.get(`/products/${item.productId}/reviews`)).summary.count

  const pendingRow = page
    .locator('li')
    .filter({ has: page.getByRole('link', { name: delivered.orderNumber }) })
    .filter({ has: page.getByRole('button', { name: 'Beri ulasan' }) })

  await test.step('/account lists the delivered item under "Menunggu ulasan"', async () => {
    await page.goto('/account')
    await expect(page.getByRole('heading', { name: 'Menunggu ulasan' })).toBeVisible()
    await expect(pendingRow).toBeVisible()
  })

  const dialog = page.getByRole('dialog', { name: 'Beri ulasan' })
  await test.step('the dialog refuses a review without stars', async () => {
    await pendingRow.getByRole('button', { name: 'Beri ulasan' }).click()
    await dialog.getByRole('button', { name: 'Kirim ulasan' }).click()
    await expect(dialog.getByText('Pilih 1 sampai 5 bintang')).toBeVisible()
  })

  await test.step('4 stars + comment -> saved, toast, moves to "Ulasan saya"', async () => {
    await dialog.getByRole('radio', { name: /^4 bintang/ }).locator('..').click() // the star the user clicks
    await expect(dialog.getByText('Bagus', { exact: true })).toBeVisible()
    await dialog.getByLabel('Ulasan (opsional)').fill(COMMENT)
    await expect(dialog.getByText(`${[...COMMENT].length}/1000 karakter`)).toBeVisible()
    await dialog.getByRole('button', { name: 'Kirim ulasan' }).click()
    await expect(page.getByText('Terima kasih! Ulasanmu sudah tersimpan.')).toBeVisible()
    await expect(dialog).toBeHidden()
    await expect(page.getByText(COMMENT)).toBeVisible()
    await expect(pendingRow).toHaveCount(0)
  })

  await test.step('the product page shows it with a masked name and the summary +1, no email', async () => {
    await page.goto(CABLE)
    const section = page.locator('section#ulasan')
    await expect(section.getByRole('heading', { name: 'Ulasan pembeli' })).toBeVisible()
    // exact: the distribution bars also say "4 bintang: 1 ulasan" (screen-reader text).
    await expect(section.getByText(`${before + 1} ulasan`, { exact: true })).toBeVisible()
    const row = section.locator('li').filter({ hasText: COMMENT })
    await expect(row.getByText('Budi P***')).toBeVisible()
    await expect(row.getByRole('img', { name: '4,0 dari 5 bintang' })).toBeVisible()
    expect(await section.innerText()).not.toContain('@')
  })

  await test.step('the rating next to the title links to the reviews', async () => {
    await page.getByRole('link', { name: /\(\d+\)/ }).first().click()
    await expect(page).toHaveURL(/#ulasan$/)
  })

  await test.step('the order page opens the review prefilled; edit to 2 stars', async () => {
    await page.goto(`/orders/${delivered.id}`)
    await page.getByRole('button', { name: 'Ubah ulasan' }).click()
    const edit = page.getByRole('dialog', { name: 'Ubah ulasan' })
    await expect(edit.getByLabel('Ulasan (opsional)')).toHaveValue(COMMENT)
    await expect(edit.getByRole('radio', { name: /^4 bintang/ })).toBeChecked()
    await edit.getByRole('radio', { name: /^2 bintang/ }).locator('..').click()
    await expect(edit.getByRole('radio', { name: /^2 bintang/ })).toBeChecked()
    await edit.getByLabel('Ulasan (opsional)').fill(EDITED)
    await edit.getByRole('button', { name: 'Simpan perubahan' }).click()
    await expect(page.getByText('Ulasan diperbarui.')).toBeVisible()
    await expect(page.getByRole('img', { name: '2,0 dari 5 bintang' })).toBeVisible()
  })

  await test.step('the product page shows the edit marker and the new text', async () => {
    await page.goto(`${CABLE}#ulasan`)
    const row = page.locator('section#ulasan li').filter({ hasText: EDITED })
    await expect(row.getByText(/· diubah$/)).toBeVisible()
    await expect(row.getByRole('img', { name: '2,0 dari 5 bintang' })).toBeVisible()
  })

  await test.step('an undelivered (paid) order has no review button', async () => {
    await page.goto(`/orders/${undelivered.id}`)
    await expect(page.getByRole('heading', { name: undelivered.orderNumber })).toBeVisible()
    await expect(page.getByRole('button', { name: /ulasan/ })).toHaveCount(0)
  })

  await test.step('"Rating tertinggi" puts reviewed products first', async () => {
    await page.goto('/products')
    await page.getByRole('combobox', { name: 'Urutkan' }).click()
    await page.getByRole('option', { name: 'Rating tertinggi' }).click()
    await page.waitForURL((u) => u.searchParams.get('sort') === 'rating')
    const top = (await api.get('/products?sort=rating&pageSize=1')).items[0]
    expect(top.rating.count).toBeGreaterThan(0)
    await expect(page.locator('main h3').first()).toHaveText(top.name)
  })

  await test.step('the API refuses a second review of the same item (409)', async () => {
    const res = await api.call('POST', '/reviews', { orderItemId: item.id, rating: 5, comment: '' })
    expect(res.status).toBe(409)
    expect(res.body.error.code).toBe('REVIEW_EXISTS')
  })
})
