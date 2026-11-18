import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../lib/api'
import { applyServerError, messageFor } from './serverErrors'

describe('messageFor', () => {
  it('translates known codes', () => {
    expect(messageFor(new ApiError(401, 'INVALID_CREDENTIALS', 'invalid credentials'))).toBe('Email atau kata sandi salah.')
    expect(messageFor(new ApiError(409, 'OWN_PRODUCT', 'own'))).toBe('Ini produk dari tokomu sendiri.')
  })

  it('falls back to the server message for unknown codes', () => {
    expect(messageFor(new ApiError(500, 'SOMETHING_NEW', 'Server said this'))).toBe('Server said this')
  })

  it('explains stock limits with the numbers from the details', () => {
    expect(messageFor(new ApiError(409, 'QUANTITY_EXCEEDS_STOCK', 'x', { available: 3, inCart: 2 }))).toBe(
      'Stok tersisa 3, dan 2 sudah ada di keranjangmu.',
    )
    expect(messageFor(new ApiError(409, 'QUANTITY_EXCEEDS_STOCK', 'x', { available: 3, inCart: 0 }))).toBe('Stok tersisa 3.')
  })

  it('never shows raw details of a non-API error', () => {
    expect(messageFor(new Error('TypeError: x is undefined'))).toBe('Terjadi kesalahan tak terduga.')
  })
})

describe('applyServerError', () => {
  it('puts translated field errors on the form, with nested paths converted', () => {
    const setError = vi.fn()
    const err = new ApiError(422, 'VALIDATION_ERROR', 'invalid', {
      email: 'is already registered',
      'variants[0].price': 'may have at most 2 decimal places',
    })

    const formMessage = applyServerError(err, setError, ['email', 'variants'] as never)

    expect(formMessage).toBeNull() // fields explain it; no duplicate message above the form
    expect(setError).toHaveBeenCalledWith('email', { type: 'server', message: 'Sudah terdaftar' })
    expect(setError).toHaveBeenCalledWith('variants.0.price', { type: 'server', message: 'Maksimal 2 angka desimal' })
  })

  it('returns a form-level message when no field matched', () => {
    const setError = vi.fn()
    const err = new ApiError(409, 'SLUG_TAKEN', 'taken', { unknownField: 'is already used' })

    expect(applyServerError(err, setError, ['name'] as never)).toBe('Nama ini sudah dipakai. Coba nama lain.')
    expect(setError).not.toHaveBeenCalled()
  })

  it('keeps untranslated server text as it is (capitalised)', () => {
    const setError = vi.fn()
    applyServerError(new ApiError(422, 'VALIDATION_ERROR', 'x', { name: 'brand new rule' }), setError, ['name'] as never)
    expect(setError).toHaveBeenCalledWith('name', { type: 'server', message: 'Brand new rule' })
  })
})
