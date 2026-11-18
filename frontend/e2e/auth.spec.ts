// Phase 1: register, login, logout, session cookie, guard, profile.
import { uniqueEmail } from './support/data'
import { expect, logoutUI, test } from './support/fixtures'

test('sign up, profile, logout, login, duplicate email', async ({ page }) => {
  const email = uniqueEmail('auth')
  const password = 'rahasia-e2e-123'

  await test.step('anonymous /account redirects to /login?next=/account', async () => {
    await page.goto('/account')
    await page.waitForURL(/\/login\?next=%2Faccount/)
  })

  await test.step('short password is rejected in the browser', async () => {
    await page.getByRole('link', { name: 'Daftar' }).last().click()
    await page.waitForURL(/\/register/)
    await page.getByLabel('Nama').fill('Pembeli E2E')
    await page.getByLabel('Email').fill(email)
    await page.getByLabel('Kata sandi').fill('pendek')
    await page.getByRole('button', { name: 'Daftar' }).click()
    await expect(page.getByText('Kata sandi minimal 8 karakter')).toBeVisible()
  })

  await test.step('register logs in and returns to /account', async () => {
    await page.getByLabel('Kata sandi').fill(password)
    await page.getByRole('button', { name: 'Daftar' }).click()
    await page.waitForURL(/\/account$/)
    await expect(page.getByRole('heading', { name: 'Akun saya' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Pembeli E2E' })).toBeVisible()
    await expect(page.getByText(email)).toBeVisible()
    await expect(page.getByText('Pembeli', { exact: true })).toBeVisible()
  })

  await test.step('the session cookie is HttpOnly + SameSite=Lax and invisible to JavaScript', async () => {
    const sid = (await page.context().cookies()).find((c) => c.name === 'sid')
    expect(sid?.httpOnly).toBe(true)
    expect(sid?.sameSite).toBe('Lax')
    expect(await page.evaluate(() => document.cookie)).not.toContain('sid=')
  })

  await test.step('profile: invalid phone rejected, valid phone saved and kept after reload', async () => {
    await page.getByLabel('Nomor telepon').fill('abc')
    await page.getByRole('button', { name: 'Simpan' }).click()
    await expect(page.getByText('Hanya angka, spasi, dan + - ( )')).toBeVisible()
    await page.getByLabel('Nomor telepon').fill('+62 812 3333 4444')
    await page.getByRole('button', { name: 'Simpan' }).click()
    await expect(page.getByText('Profil tersimpan.')).toBeVisible()
    await page.reload()
    await expect(page.getByLabel('Nomor telepon')).toHaveValue('+62 812 3333 4444')
  })

  await test.step('logout returns home; /account needs a login again', async () => {
    await logoutUI(page)
    await page.goto('/account')
    await page.waitForURL(/\/login/)
  })

  await test.step('wrong password shows a generic error; the right one (upper-case email) honours ?next', async () => {
    await page.getByLabel('Email').fill(email.toUpperCase())
    await page.getByLabel('Kata sandi').fill('salah-salah-salah')
    await page.getByRole('button', { name: 'Masuk' }).click()
    await expect(page.getByText('Email atau kata sandi salah.')).toBeVisible()
    await page.getByLabel('Kata sandi').fill(password)
    await page.getByRole('button', { name: 'Masuk' }).click()
    await page.waitForURL(/\/account$/)
  })

  await test.step('a duplicate email is shown on the email field', async () => {
    await logoutUI(page)
    await page.goto('/register')
    await page.getByLabel('Nama').fill('Orang Lain')
    await page.getByLabel('Email').fill(email)
    await page.getByLabel('Kata sandi').fill('rahasia-lain-123')
    await page.getByRole('button', { name: 'Daftar' }).click()
    await expect(page.getByText('Sudah terdaftar')).toBeVisible()
  })
})
