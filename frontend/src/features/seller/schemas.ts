// Client-side form rules mirroring backend/internal/modules/{stores,products}.
// They give instant feedback; the server re-checks everything.
import { z } from 'zod'

export const PRICE_PATTERN = /^\d{1,10}(\.\d{1,2})?$/
export const SKU_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/
export const MAX_STOCK = 1_000_000

export const priceField = z
  .string()
  .trim()
  .regex(PRICE_PATTERN, 'Harga berupa angka, maksimal 2 desimal (contoh 349000)')

export const stockField = z
  .string()
  .trim()
  .regex(/^\d+$/, 'Stok berupa bilangan bulat ≥ 0')
  .refine((v) => Number(v) <= MAX_STOCK, `Stok maksimal ${MAX_STOCK.toLocaleString('id-ID')}`)

export const variantSchema = z.object({
  name: z.string().trim().min(1, 'Nama varian wajib diisi').max(100, 'Maksimal 100 karakter'),
  sku: z.string().trim().regex(SKU_PATTERN, 'Huruf, angka, . _ - (maks 64)'),
  price: priceField,
  stock: stockField,
})

/** One http(s) URL per line; blank lines are ignored. */
export const imagesField = z
  .string()
  .refine((v) => parseImageLines(v).length <= 8, 'Maksimal 8 gambar')
  .refine(
    (v) => parseImageLines(v).every((u) => /^https?:\/\/[^\s]+$/i.test(u)),
    'Setiap baris harus URL yang diawali http:// atau https://',
  )

export function parseImageLines(text: string): string[] {
  return text
    .split('\n')
    .map((l) => l.trim())
    .filter(Boolean)
}

export const productSchema = z.object({
  name: z.string().trim().min(3, 'Minimal 3 karakter').max(200, 'Maksimal 200 karakter'),
  categoryId: z.string().min(1, 'Pilih kategori'),
  description: z.string().trim().max(5000, 'Maksimal 5000 karakter'),
  images: imagesField,
})

export const newProductSchema = productSchema.extend({
  variants: z
    .array(variantSchema)
    .min(1, 'Minimal satu varian')
    .max(20, 'Maksimal 20 varian')
    .refine((vs) => new Set(vs.map((v) => v.sku.trim().toUpperCase())).size === vs.length, 'SKU tiap varian harus berbeda'),
})

export const openStoreSchema = z.object({
  name: z.string().trim().min(3, 'Minimal 3 karakter').max(100, 'Maksimal 100 karakter'),
  location: z.string().trim().min(2, 'Isi kota atau kabupaten').max(100, 'Maksimal 100 karakter'),
  description: z.string().trim().max(1000, 'Maksimal 1000 karakter'),
})

export type ProductValues = z.infer<typeof productSchema>
export type NewProductValues = z.infer<typeof newProductSchema>
export type VariantValues = z.infer<typeof variantSchema>
export type OpenStoreValues = z.infer<typeof openStoreSchema>
