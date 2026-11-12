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
}

const CODE_MESSAGES: Record<string, string> = {
  INVALID_CREDENTIALS: 'Email atau kata sandi salah.',
  ACCOUNT_SUSPENDED: 'Akun ini sedang ditangguhkan. Hubungi admin.',
  RATE_LIMITED: 'Terlalu banyak percobaan. Coba lagi beberapa menit lagi.',
  EMAIL_TAKEN: 'Email ini sudah terdaftar.',
  NETWORK_ERROR: 'Tidak dapat terhubung ke server. Pastikan backend berjalan.',
  ORIGIN_REJECTED: 'Permintaan ditolak karena berasal dari situs lain.',
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
    if ((fields as readonly string[]).includes(field)) {
      const text = FIELD_MESSAGES[message] ?? message
      setError(field as Path<T>, { type: 'server', message: text.charAt(0).toUpperCase() + text.slice(1) })
      placed++
    }
  }
  // Field messages already explain the problem; don't repeat it above the form.
  if (placed > 0) return null
  return CODE_MESSAGES[err.code] ?? err.message
}
