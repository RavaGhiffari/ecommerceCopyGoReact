// Phase 5a: the address book on /account, for a fresh customer.
import type { Locator, Page } from '@playwright/test'
import { expect, test } from './support/fixtures'

async function fillAddress(dialog: Locator, v: { label: string; city?: string; postal?: string }) {
  await dialog.getByLabel('Label').fill(v.label)
  await dialog.getByLabel('Nama penerima').fill('Budi Pembeli')
  await dialog.getByLabel('Nomor telepon').fill('0812-3456-7890')
  await dialog.getByLabel('Alamat lengkap').fill('Jl. Kenanga No. 12, RT 03/RW 04')
  await dialog.getByLabel('Kota/kabupaten').fill(v.city ?? 'Bandung')
  await dialog.getByLabel('Provinsi').fill('Jawa Barat')
  await dialog.getByLabel('Kode pos').fill(v.postal ?? '40115')
}

const address = (page: Page, label: string) => page.locator(`article[aria-label="Alamat ${label}"]`)

test('address book: create, default rules, edit, delete with confirmation', async ({ page, customer }) => {
  void customer // signed in by the fixture
  const items = page.locator('article[aria-label^="Alamat "]')
  const dialog = page.getByRole('dialog')

  await page.goto('/account')
  await expect(page.getByRole('heading', { name: 'Alamat pengiriman' })).toBeVisible()

  await test.step('a user without addresses sees the empty state', async () => {
    await expect(page.getByText('Belum ada alamat')).toBeVisible()
  })

  await test.step('client validation blocks an invalid form', async () => {
    await page.getByRole('button', { name: 'Tambah alamat' }).click()
    await expect(dialog.getByRole('heading', { name: 'Tambah alamat' })).toBeVisible()
    await dialog.getByLabel('Kode pos').fill('12')
    await dialog.getByRole('button', { name: 'Simpan alamat' }).click()
    await expect(dialog.getByText('Kode pos harus 5 digit angka')).toBeVisible()
    await expect(dialog.getByText('Label wajib diisi')).toBeVisible()
    await expect(dialog.getByText('Alamat pertama otomatis menjadi alamat utama.')).toBeVisible()
  })

  await test.step('the first address becomes the default; the phone is normalised', async () => {
    await fillAddress(dialog, { label: 'Rumah' })
    await dialog.getByRole('button', { name: 'Simpan alamat' }).click()
    await expect(dialog).toBeHidden()
    await expect(items).toHaveCount(1)
    await expect(items.first().getByText('Utama')).toBeVisible()
    await expect(items.first().getByText('081234567890')).toBeVisible()
  })

  await test.step('a second address with "Jadikan alamat utama" takes the default and is listed first', async () => {
    await page.getByRole('button', { name: 'Tambah alamat' }).click()
    await fillAddress(dialog, { label: 'Kantor', city: 'Jakarta Selatan', postal: '12160' })
    await dialog.getByLabel('Jadikan alamat utama').check()
    await dialog.getByRole('button', { name: 'Simpan alamat' }).click()
    await expect(dialog).toBeHidden()
    await expect(address(page, 'Kantor').getByText('Utama')).toBeVisible()
    await expect(items).toHaveCount(2)
    await expect(items.first()).toHaveAttribute('aria-label', 'Alamat Kantor')
  })

  await test.step('"Jadikan utama" moves the default back; only one badge', async () => {
    await address(page, 'Rumah').getByRole('button', { name: 'Jadikan utama' }).click()
    await expect(address(page, 'Rumah').getByText('Utama', { exact: true })).toBeVisible()
    await expect(page.getByText('Utama', { exact: true })).toHaveCount(1)
  })

  await test.step('edit opens prefilled and saves the change', async () => {
    await address(page, 'Kantor').getByRole('button', { name: 'Ubah' }).click()
    await expect(dialog.getByLabel('Kota/kabupaten')).toHaveValue('Jakarta Selatan')
    await dialog.getByLabel('Kota/kabupaten').fill('Depok')
    await dialog.getByRole('button', { name: 'Simpan alamat' }).click()
    await expect(dialog).toBeHidden()
    await expect(address(page, 'Kantor').getByText(/Depok/)).toBeVisible()
  })

  await test.step('delete asks first; "Batal" keeps the address', async () => {
    await address(page, 'Rumah').getByRole('button', { name: 'Hapus' }).click()
    const confirm = page.getByRole('alertdialog')
    await expect(confirm.getByText('Alamat terbaru lainnya akan menjadi alamat utama.')).toBeVisible()
    await confirm.getByRole('button', { name: 'Batal' }).click()
    await expect(confirm).toBeHidden()
    await expect(items).toHaveCount(2)
  })

  await test.step('deleting the default promotes the remaining address', async () => {
    await address(page, 'Rumah').getByRole('button', { name: 'Hapus' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Hapus' }).click()
    await expect(address(page, 'Rumah')).toHaveCount(0)
    await expect(address(page, 'Kantor').getByText('Utama')).toBeVisible()
  })

  await test.step('the list comes from the server (survives a reload)', async () => {
    await page.reload()
    await expect(address(page, 'Kantor')).toBeVisible()
    await expect(items).toHaveCount(1)
  })

  await test.step('deleting the last address shows the empty state again', async () => {
    await address(page, 'Kantor').getByRole('button', { name: 'Hapus' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Hapus' }).click()
    await expect(page.getByText('Belum ada alamat')).toBeVisible()
  })
})
