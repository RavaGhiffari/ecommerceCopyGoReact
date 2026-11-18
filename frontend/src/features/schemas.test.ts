// The zod schemas mirror backend validation rules for instant feedback.
// These tests pin the edges (lengths, patterns) so a change on either side
// is noticed. The server stays the authority.
import { describe, expect, it } from 'vitest'
import type { z } from 'zod'
import { addressSchema, normalizePhone } from './addresses/schema'
import { categorySchema } from './admin/categorySchema'
import { profileSchema, registerSchema } from './auth/schemas'
import { charCount, reviewSchema } from './reviews/schema'
import { newProductSchema, parseImageLines } from './seller/schemas'

/**
 * The first message of every failing field, keyed by path ("variants.0.sku").
 * The first one is what the form shows (React Hook Form keeps the first issue).
 */
function errorsOf(schema: z.ZodType, value: unknown): Record<string, string> {
  const r = schema.safeParse(value)
  if (r.success) return {}
  const out: Record<string, string> = {}
  for (const i of r.error.issues) out[i.path.join('.')] ??= i.message
  return out
}

describe('registerSchema', () => {
  const ok = { name: 'Agus', email: 'agus@contoh.id', password: 'rahasia-123' }

  it('accepts a valid sign-up', () => {
    expect(errorsOf(registerSchema, ok)).toEqual({})
  })

  it('counts the password length: 8 to 128', () => {
    expect(errorsOf(registerSchema, { ...ok, password: '1234567' })).toEqual({ password: 'Kata sandi minimal 8 karakter' })
    expect(errorsOf(registerSchema, { ...ok, password: 'x'.repeat(129) })).toHaveProperty('password')
    expect(errorsOf(registerSchema, { ...ok, password: 'x'.repeat(128) })).toEqual({})
  })

  it('rejects a malformed email and a blank name', () => {
    expect(Object.keys(errorsOf(registerSchema, { ...ok, email: 'nope', name: '   ' })).sort()).toEqual(['email', 'name'])
  })
})

describe('profileSchema', () => {
  it.each([
    ['', true],
    ['+62 812 3333 4444', true],
    ['abc', false],
    ['1234567', false], // fewer than 8 digits
  ])('phone %j valid=%s', (phone, valid) => {
    expect(profileSchema.safeParse({ name: 'Agus', phone }).success).toBe(valid)
  })
})

describe('addressSchema', () => {
  const ok = {
    label: 'Rumah',
    recipientName: 'Agus',
    phone: '0812-3456-7890',
    addressLine: 'Jl. Kenanga No. 12',
    city: 'Bandung',
    province: 'Jawa Barat',
    postalCode: '40115',
    isDefault: false,
  }

  it('accepts a valid address', () => {
    expect(errorsOf(addressSchema, ok)).toEqual({})
  })

  it('normalises the phone the way the server does', () => {
    expect(normalizePhone(' (0812) 3456-7890 ')).toBe('081234567890')
    expect(errorsOf(addressSchema, { ...ok, phone: '0812' })).toHaveProperty('phone')
  })

  it('requires a 5-digit postal code and a label', () => {
    expect(errorsOf(addressSchema, { ...ok, postalCode: '12', label: '' })).toEqual({
      postalCode: 'Kode pos harus 5 digit angka',
      label: 'Label wajib diisi',
    })
  })
})

describe('reviewSchema', () => {
  it('needs 1 to 5 stars', () => {
    expect(errorsOf(reviewSchema, { rating: 0, comment: '' })).toEqual({ rating: 'Pilih 1 sampai 5 bintang' })
    expect(errorsOf(reviewSchema, { rating: 6, comment: '' })).toHaveProperty('rating')
    expect(errorsOf(reviewSchema, { rating: 5, comment: '' })).toEqual({})
  })

  it('counts characters like the server (code points, not UTF-16 units)', () => {
    const emoji = '😀'.repeat(1000) // 2000 UTF-16 units, 1000 characters
    expect(charCount(emoji)).toBe(1000)
    expect(errorsOf(reviewSchema, { rating: 4, comment: emoji })).toEqual({})
    expect(errorsOf(reviewSchema, { rating: 4, comment: emoji + 'x' })).toHaveProperty('comment')
  })
})

describe('categorySchema', () => {
  const ok = { name: 'Audio', slug: '', parentId: 'none', sortOrder: '0' }

  it.each([
    ['', true], // empty = derived by the server
    ['audio-mobil', true],
    ['Bad Slug', false],
    ['double--hyphen', false],
    ['-leading', false],
  ])('slug %j valid=%s', (slug, valid) => {
    expect(categorySchema.safeParse({ ...ok, slug }).success).toBe(valid)
  })

  it('needs an integer sort order', () => {
    expect(errorsOf(categorySchema, { ...ok, sortOrder: '1.5' })).toEqual({ sortOrder: 'Isi bilangan bulat' })
    expect(categorySchema.safeParse({ ...ok, sortOrder: '-3' }).success).toBe(true)
  })
})

describe('newProductSchema', () => {
  const variant = { name: 'Hitam', sku: 'UJI-1', price: '125000', stock: '7' }
  const ok = { name: 'Headphone Uji', categoryId: 'c-1', description: '', images: '', variants: [variant] }

  it('accepts a valid product', () => {
    expect(errorsOf(newProductSchema, ok)).toEqual({})
  })

  it('rejects a comma price (the E2E case "12,5") and more than 2 decimals', () => {
    expect(errorsOf(newProductSchema, { ...ok, variants: [{ ...variant, price: '12,5' }] })).toHaveProperty('variants.0.price')
    expect(errorsOf(newProductSchema, { ...ok, variants: [{ ...variant, price: '1.005' }] })).toHaveProperty('variants.0.price')
    expect(errorsOf(newProductSchema, { ...ok, variants: [{ ...variant, price: '1.5' }] })).toEqual({})
  })

  it('refuses duplicate SKUs ignoring case', () => {
    const errs = errorsOf(newProductSchema, { ...ok, variants: [variant, { ...variant, name: 'Putih', sku: 'uji-1' }] })
    expect(errs).toEqual({ variants: 'SKU tiap varian harus berbeda' })
  })

  it('needs 1 to 20 variants', () => {
    expect(errorsOf(newProductSchema, { ...ok, variants: [] })).toEqual({ variants: 'Minimal satu varian' })
    const many = Array.from({ length: 21 }, (_, i) => ({ ...variant, sku: `S-${i}` }))
    expect(errorsOf(newProductSchema, { ...ok, variants: many })).toEqual({ variants: 'Maksimal 20 varian' })
  })

  it('takes one http(s) image URL per line, at most 8', () => {
    expect(parseImageLines(' https://a.example/1.jpg \n\n https://a.example/2.jpg ')).toEqual([
      'https://a.example/1.jpg',
      'https://a.example/2.jpg',
    ])
    expect(errorsOf(newProductSchema, { ...ok, images: 'ftp://a.example/x.jpg' })).toHaveProperty('images')
    const nine = Array.from({ length: 9 }, (_, i) => `https://a.example/${i}.jpg`).join('\n')
    expect(errorsOf(newProductSchema, { ...ok, images: nine })).toEqual({ images: 'Maksimal 8 gambar' })
  })
})
