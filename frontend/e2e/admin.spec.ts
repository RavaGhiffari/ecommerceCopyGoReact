// Phase 6e: admin panel. The admin (e2e-admin@contoh.id) is created by
// `make e2e-reset` with a random password, written to e2e/.auth/admin-password.
// Every change is reverted: suspend -> reactivate, hide -> show, category
// sort order -> back. Only audit-log rows stay (append-only by design).
import { existsSync, readFileSync } from 'node:fs'
import type { Page } from '@playwright/test'
import { ADMIN_EMAIL, SELLER_BATIK, SELLER_SPORTIVO } from './support/data'
import { expect, loginUI, test } from './support/fixtures'

const PASSWORD_FILE = new URL('./.auth/admin-password', import.meta.url)
const ADMIN_PASSWORD =
  process.env.E2E_ADMIN_PASSWORD ?? (existsSync(PASSWORD_FILE) ? readFileSync(PASSWORD_FILE, 'utf8').trim() : '')

test.skip(!ADMIN_PASSWORD, 'no admin password: run `make e2e-reset` (or set E2E_ADMIN_PASSWORD)')

const overflow = (page: Page) => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)

test('admin panel: access, dashboard, users, stores, categories, reviews, audit log', async ({ page, actor }) => {
  const admin = page

  await test.step('a non-admin sees the admin-only notice and no header link', async () => {
    const seller = await actor()
    await loginUI(seller, SELLER_BATIK)
    await expect(seller.locator('header a[href="/admin"]')).toHaveCount(0)
    await seller.goto('/admin/users')
    await expect(seller.getByRole('heading', { name: 'Halaman khusus admin' })).toBeVisible()
  })

  await test.step('anonymous /admin/stores goes to login', async () => {
    const anon = await actor()
    await anon.goto('/admin/stores')
    await anon.waitForURL((u) => u.pathname === '/login' && u.search.includes('admin'))
  })

  await test.step('admin: header link and dashboard tiles', async () => {
    await loginUI(admin, ADMIN_EMAIL, ADMIN_PASSWORD)
    await admin.locator('header a[href="/admin"]').click()
    await expect(admin.getByRole('heading', { name: 'Ringkasan platform' })).toBeVisible()
    await expect(admin.getByText('GMV 30 hari')).toBeVisible()
    await expect(admin.locator('p', { hasText: /^Rp/ }).first()).toBeVisible()
  })

  await test.step('a dashboard status link filters the order list', async () => {
    await admin.getByRole('link', { name: /Selesai/ }).first().click()
    await admin.waitForURL((u) => u.pathname === '/admin/orders' && u.search === '?status=delivered')
    await expect(admin.getByRole('button', { name: 'Selesai', pressed: true })).toBeVisible()
    await expect(admin.locator('tbody tr').first()).toBeVisible()
  })

  await test.step('order detail opens read-only', async () => {
    await admin.locator('tbody tr').first().getByRole('link', { name: /^Detail/ }).click()
    await admin.waitForURL((u) => u.pathname.startsWith('/admin/orders/'))
    await expect(admin.getByRole('heading', { level: 1, name: /^PK-/ })).toBeVisible()
    await expect(admin.getByText('Admin hanya bisa melihat pesanan.')).toBeVisible()
  })

  const userRow = admin.locator('tbody tr', { hasText: SELLER_SPORTIVO })

  await test.step('users: search; the own row has no action', async () => {
    await admin.goto('/admin/users?q=e2e-admin')
    const me = admin.locator('tbody tr', { hasText: ADMIN_EMAIL })
    await expect(me.getByText('(kamu)')).toHaveCount(1)
    await expect(me.getByRole('button')).toHaveCount(0)
    await admin.getByRole('searchbox', { name: 'Cari pengguna' }).fill('penjual4')
    await admin.getByRole('button', { name: 'Cari pengguna: cari' }).click()
    await admin.waitForURL((u) => u.search === '?q=penjual4')
    await expect(userRow).toBeVisible()
  })

  await test.step('users: suspend asks first, then ends the user’s open session at once', async () => {
    const victim = await actor()
    await loginUI(victim, SELLER_SPORTIVO)
    await userRow.getByRole('button', { name: 'Tangguhkan' }).click()
    const dlg = admin.getByRole('alertdialog')
    await expect(dlg.getByText('langsung keluar dari semua perangkat')).toBeVisible()
    await dlg.getByRole('button', { name: 'Batal' }).click()
    await expect(userRow.getByText('Aktif')).toHaveCount(1)
    await userRow.getByRole('button', { name: 'Tangguhkan' }).click()
    await admin.getByRole('alertdialog').getByRole('button', { name: 'Tangguhkan' }).click()
    await expect(userRow.getByText('Ditangguhkan')).toBeVisible()
    const status = await victim.evaluate(async () => (await fetch('/api/v1/me', { credentials: 'include' })).status)
    expect(status, 'the suspended user’s session still works').toBe(401)
  })

  await test.step('users: reactivate', async () => {
    await userRow.getByRole('button', { name: 'Aktifkan' }).click()
    await expect(userRow.getByText('Aktif', { exact: true })).toBeVisible()
  })

  await test.step('stores: suspending hides the store, reactivating brings it back', async () => {
    await admin.goto('/admin/stores?q=penjual4')
    const storeRow = admin.locator('tbody tr', { hasText: SELLER_SPORTIVO })
    const href = await storeRow.locator('a[href^="/stores/"]').getAttribute('href')
    const slug = href!.split('/')[2]
    const publicStatus = () => admin.evaluate(async (s) => (await fetch(`/api/v1/stores/${s}`)).status, slug)

    await storeRow.getByRole('button', { name: 'Tangguhkan' }).click()
    await expect(admin.getByRole('alertdialog').getByText('Pesanan yang sudah ada tetap berjalan')).toBeVisible()
    await admin.getByRole('alertdialog').getByRole('button', { name: 'Tangguhkan toko' }).click()
    await expect(storeRow.getByText('Ditangguhkan')).toBeVisible()
    await expect(storeRow.locator('a[href^="/stores/"]')).toHaveCount(0)
    expect(await publicStatus()).toBe(404)

    await storeRow.getByRole('button', { name: 'Aktifkan' }).click()
    await expect(storeRow.getByText('Aktif', { exact: true })).toBeVisible()
    expect(await publicStatus()).toBe(200)
  })

  await test.step('stores: the status filter comes from the URL', async () => {
    await admin.goto('/admin/stores?status=suspended')
    await expect(admin.getByRole('button', { name: 'Ditangguhkan', pressed: true })).toBeVisible()
  })

  await test.step('categories: edit and revert', async () => {
    await admin.goto('/admin/categories')
    const first = admin.locator('tbody tr').first()
    // Wait for a real row: while loading, the table shows skeleton rows with no text.
    await expect(first.getByRole('button', { name: /^Ubah/ })).toBeVisible()
    const name = (await first.locator('td').first().innerText()).trim()
    const order = (await first.locator('td').nth(2).innerText()).trim()
    const row = admin.locator('tbody tr', { hasText: name }).first()
    const dlg = admin.getByRole('dialog')

    await row.getByRole('button', { name: /^Ubah/ }).click()
    await expect(dlg.getByLabel('Nama')).toHaveValue(name)
    await dlg.getByLabel('Urutan').fill('777')
    await dlg.getByRole('button', { name: 'Simpan perubahan' }).click()
    await expect(dlg).toBeHidden()
    await expect(row.getByText('777')).toBeVisible()

    await row.getByRole('button', { name: /^Ubah/ }).click()
    await dlg.getByLabel('Urutan').fill(order)
    await dlg.getByRole('button', { name: 'Simpan perubahan' }).click()
    await expect(dlg).toBeHidden()
  })

  await test.step('categories: client and server validation; nothing is created', async () => {
    const rows = admin.locator('tbody tr')
    const before = await rows.count()
    const takenSlug = (await rows.first().locator('td').nth(1).innerText()).trim()
    await admin.getByRole('button', { name: 'Tambah kategori' }).click()
    const dlg = admin.getByRole('dialog')
    await dlg.getByRole('button', { name: 'Tambah kategori' }).click()
    await expect(dlg.getByText('Nama wajib diisi')).toBeVisible()
    await dlg.getByLabel('Slug').fill('Bad Slug')
    await dlg.getByLabel('Nama').fill('Duplikat E2E')
    await dlg.getByRole('button', { name: 'Tambah kategori' }).click()
    await expect(dlg.getByText('Hanya a–z, 0–9')).toBeVisible()
    await dlg.getByLabel('Slug').fill(takenSlug) // the server answers 409; the message lands on the field
    await dlg.getByRole('button', { name: 'Tambah kategori' }).click()
    await expect(dlg.getByText('Sudah dipakai')).toBeVisible()
    await admin.keyboard.press('Escape')
    await expect(dlg).toBeHidden()
    await expect(rows).toHaveCount(before)
  })

  await test.step('reviews: hide and show again', async () => {
    await admin.goto('/admin/reviews?status=visible')
    const first = admin.locator('ul > li article').first()
    await first.getByRole('button', { name: 'Sembunyikan' }).click()
    await admin.getByRole('alertdialog').getByRole('button', { name: 'Sembunyikan' }).click()
    await expect(first.getByText('Disembunyikan')).toBeVisible()
    await first.getByRole('button', { name: 'Tampilkan' }).click()
    await expect(first.getByRole('button', { name: 'Sembunyikan' })).toBeVisible()
  })

  await test.step('the audit log lists this run', async () => {
    await admin.goto('/admin/audit-logs')
    await expect(admin.locator('tbody tr').first()).toBeVisible()
    for (const label of [
      'Menangguhkan pengguna',
      'Mengaktifkan pengguna',
      'Menangguhkan toko',
      'Mengaktifkan toko',
      'Mengubah kategori',
      'Menyembunyikan ulasan',
      'Menampilkan ulasan',
    ]) {
      await expect(admin.getByText(label, { exact: true }).first(), label).toBeVisible()
    }
    await expect(admin.getByText(/sesi diakhiri/).first()).toBeVisible()
  })

  await test.step('390 px: no horizontal page scroll on users and reviews', async () => {
    const phone = await actor({ width: 390 })
    await loginUI(phone, ADMIN_EMAIL, ADMIN_PASSWORD, '/admin/users')
    await expect(phone.locator('tbody tr').first()).toBeVisible()
    expect(await overflow(phone), 'users page overflows').toBeLessThanOrEqual(0)
    await phone.goto('/admin/reviews')
    await expect(phone.locator('ul > li article').first()).toBeVisible()
    expect(await overflow(phone), 'reviews page overflows').toBeLessThanOrEqual(0)
  })
})
