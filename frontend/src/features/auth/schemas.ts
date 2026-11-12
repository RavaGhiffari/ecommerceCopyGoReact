// Client-side form rules. They mirror the backend rules
// (backend/internal/modules/auth/service.go, users/model.go) to give instant
// feedback, but the server re-checks everything and is the only authority.
import { z } from 'zod'

const email = z
  .string()
  .trim()
  .min(1, 'Email wajib diisi')
  .max(255, 'Email maksimal 255 karakter')
  .pipe(z.email('Format email tidak valid'))

const name = z.string().trim().min(1, 'Nama wajib diisi').max(150, 'Nama maksimal 150 karakter')

export const loginSchema = z.object({
  email,
  password: z.string().min(1, 'Kata sandi wajib diisi'),
})

export const registerSchema = z.object({
  name,
  email,
  password: z
    .string()
    .min(8, 'Kata sandi minimal 8 karakter')
    .max(128, 'Kata sandi maksimal 128 karakter'),
})

export const profileSchema = z.object({
  name,
  phone: z
    .string()
    .trim()
    .max(30, 'Nomor telepon maksimal 30 karakter')
    .refine((v) => v === '' || /^[0-9+\-() ]+$/.test(v), 'Hanya angka, spasi, dan + - ( )')
    .refine((v) => v === '' || (v.match(/[0-9]/g) ?? []).length >= 8, 'Minimal 8 digit angka'),
})

export type LoginValues = z.infer<typeof loginSchema>
export type RegisterValues = z.infer<typeof registerSchema>
export type ProfileValues = z.infer<typeof profileSchema>
