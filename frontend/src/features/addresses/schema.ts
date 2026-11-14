// Client-side rules mirroring backend/internal/modules/addresses/model.go
// (Fields.Clean). Instant feedback only; the server re-checks everything.
import { z } from 'zod'

const text = (label: string, min: number, max: number) =>
  z
    .string()
    .trim()
    .min(1, `${label} wajib diisi`)
    .min(min, `${label} terlalu pendek`)
    .max(max, `${label} maksimal ${max} karakter`)

/** Same separators the server removes: spaces, - . ( ) */
export const normalizePhone = (v: string) => v.trim().replace(/[\s\-.()]/g, '')

export const addressSchema = z.object({
  label: text('Label', 1, 50),
  recipientName: text('Nama penerima', 2, 100),
  phone: z
    .string()
    .trim()
    .min(1, 'Nomor telepon wajib diisi')
    .refine((v) => /^\+?[0-9]{8,15}$/.test(normalizePhone(v)), '8–15 digit angka, boleh diawali +'),
  addressLine: text('Alamat', 5, 255),
  city: text('Kota/kabupaten', 2, 100),
  province: text('Provinsi', 2, 100),
  postalCode: z.string().trim().regex(/^[0-9]{5}$/, 'Kode pos harus 5 digit angka'),
  isDefault: z.boolean(),
})

export type AddressValues = z.infer<typeof addressSchema>

export const ADDRESS_FIELDS = [
  'label',
  'recipientName',
  'phone',
  'addressLine',
  'city',
  'province',
  'postalCode',
] as const
