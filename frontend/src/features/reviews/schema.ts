// Mirrors backend/internal/modules/reviews/model.go (Input.Clean). The server
// re-checks everything; this only gives instant feedback.
import { z } from 'zod'

export const MAX_COMMENT = 1000

/** Characters as the server counts them (code points, not UTF-16 units). */
export const charCount = (s: string) => [...s].length

export const reviewSchema = z.object({
  rating: z.number().int().min(1, 'Pilih 1 sampai 5 bintang').max(5, 'Pilih 1 sampai 5 bintang'),
  comment: z
    .string()
    .trim()
    .refine((v) => charCount(v) <= MAX_COMMENT, `Maksimal ${MAX_COMMENT} karakter`),
})

export type ReviewValues = z.infer<typeof reviewSchema>
