import type { FieldValues, Path, UseFormSetError } from 'react-hook-form'
import { ApiError } from '../../lib/api'

// Backend field messages are English ("is already registered"). Show them in
// Indonesian where we know them; fall back to the original text otherwise.
const FIELD_MESSAGES: Record<string, string> = {
  'is required': 'wajib diisi',
  'is already registered': 'sudah terdaftar',
  'must be a valid email address': 'format email tidak valid',
  'must be at least 8 characters': 'minimal 8 karakter',
  'must be at most 128 characters': 'maksimal 128 karakter',
  'must be at most 150 characters': 'maksimal 150 karakter',
  'is already used': 'sudah dipakai',
  'is already used in your store': 'sudah dipakai produk lain di toko kamu',
  'is used by another variant': 'dipakai varian lain',
  'is already used by another variant': 'dipakai varian lain',
  'choose a subcategory': 'pilih subkategori',
  'does not exist': 'tidak ditemukan',
  'cannot be negative': 'tidak boleh negatif',
  'may have at most 2 decimal places': 'maksimal 2 angka desimal',
  'must be an http(s) URL': 'harus URL http(s)',
}

const CODE_MESSAGES: Record<string, string> = {
  INVALID_CREDENTIALS: 'Email atau kata sandi salah.',
  ACCOUNT_SUSPENDED: 'Akun ini sedang ditangguhkan. Hubungi admin.',
  RATE_LIMITED: 'Terlalu banyak percobaan. Coba lagi beberapa menit lagi.',
  EMAIL_TAKEN: 'Email ini sudah terdaftar.',
  STORE_EXISTS: 'Kamu sudah punya toko.',
  SLUG_TAKEN: 'Nama ini sudah dipakai. Coba nama lain.',
  SKU_TAKEN: 'SKU ini sudah dipakai. Gunakan SKU lain.',
  STORE_REQUIRED: 'Buka toko terlebih dahulu.',
  FORBIDDEN: 'Kamu tidak punya akses untuk ini.',
  NETWORK_ERROR: 'Tidak dapat terhubung ke server. Pastikan backend berjalan.',
  ORIGIN_REJECTED: 'Permintaan ditolak karena berasal dari situs lain.',
  OUT_OF_STOCK: 'Stok produk ini sedang habis.',
  QUANTITY_EXCEEDS_STOCK: 'Jumlahnya melebihi stok yang tersedia.',
  VARIANT_UNAVAILABLE: 'Produk ini sudah tidak dijual.',
  VARIANT_NOT_FOUND: 'Varian produk tidak ditemukan.',
  OWN_PRODUCT: 'Ini produk dari tokomu sendiri.',
  CART_ITEM_NOT_FOUND: 'Barang ini sudah tidak ada di keranjang.',
  VALIDATION_ERROR: 'Jumlah tidak valid (1–99 per barang).',
}

/** A user-facing Indonesian message for any error from the API. */
export function messageFor(err: unknown): string {
  if (!(err instanceof ApiError)) return 'Terjadi kesalahan tak terduga.'
  if (err.code === 'QUANTITY_EXCEEDS_STOCK' && typeof err.details === 'object' && err.details !== null) {
    const d = err.details as { available?: number; inCart?: number }
    if (typeof d.available === 'number') {
      return d.inCart
        ? `Stok tersisa ${d.available}, dan ${d.inCart} sudah ada di keranjangmu.`
        : `Stok tersisa ${d.available}.`
    }
  }
  return CODE_MESSAGES[err.code] ?? err.message
}

/**
 * Puts server-side field errors onto the matching form inputs and returns a
 * message for the whole form (or null when every problem was field-level).
 */
export function applyServerError<T extends FieldValues>(
  err: unknown,
  setError: UseFormSetError<T>,
  fields: readonly Path<T>[],
): string | null {
  if (!(err instanceof ApiError)) return 'Terjadi kesalahan tak terduga.'

  let placed = 0
  for (const [field, message] of Object.entries(err.fieldErrors())) {
    // The API names nested fields like "variants[0].price"; react-hook-form
    // uses "variants.0.price". `fields` lists the allowed top-level names.
    const path = field.replace(/\[(\d+)\]/g, '.$1')
    const root = path.split('.')[0]
    if ((fields as readonly string[]).includes(root)) {
      const text = FIELD_MESSAGES[message] ?? message
      setError(path as Path<T>, { type: 'server', message: text.charAt(0).toUpperCase() + text.slice(1) })
      placed++
    }
  }
  // Field messages already explain the problem; don't repeat it above the form.
  if (placed > 0) return null
  return CODE_MESSAGES[err.code] ?? err.message
}
