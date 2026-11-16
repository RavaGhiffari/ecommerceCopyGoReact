// Client-side rules mirroring backend/internal/modules/categories/service.go
// (validate). Instant feedback only; the server re-checks everything,
// including the two-level rule and slug uniqueness.
import { z } from 'zod'

export const categorySchema = z.object({
  name: z.string().trim().min(1, 'Nama wajib diisi').max(100, 'Nama maksimal 100 karakter'),
  // Empty = the server derives it from the name (slug.Make).
  slug: z
    .string()
    .trim()
    .refine((v) => v === '' || /^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(v), 'Hanya a–z, 0–9, dan tanda hubung tunggal'),
  parentId: z.string(), // NO_PARENT = top-level
  sortOrder: z.string().trim().regex(/^-?\d{1,9}$/, 'Isi bilangan bulat'),
})

export type CategoryValues = z.infer<typeof categorySchema>

/** Radix Select cannot use "" as an item value, so "no parent" gets its own. */
export const NO_PARENT = 'none'

export const CATEGORY_FIELDS = ['name', 'slug', 'parentId', 'sortOrder'] as const
