import { Card } from '@/components/ui/card'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { FormAlert, TextField } from '../components/shared/TextField'
import { useAuth } from '../features/auth/authContext'
import { profileSchema, type ProfileValues } from '../features/auth/schemas'
import { applyServerError } from '../features/auth/serverErrors'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatMonthYear } from '../lib/format'
import type { User } from '../types/auth'
import { SubmitButton } from '../features/auth/AuthCard'

const ROLE_LABELS: Record<string, string> = { customer: 'Pembeli', seller: 'Penjual', admin: 'Admin' }

/** Rendered inside <RequireAuth>, so a user is always present. */
export function AccountPage() {
  usePageTitle('Akun saya')
  const { user } = useAuth()
  if (!user) return null
  // key: reset the form when the server returns a different user object.
  return <ProfileForm key={user.id} user={user} />
}

function ProfileForm({ user }: { user: User }) {
  const { updateProfile } = useAuth()
  const [formError, setFormError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const {
    register,
    handleSubmit,
    setError,
    reset,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<ProfileValues>({
    resolver: zodResolver(profileSchema),
    defaultValues: { name: user.name, phone: user.phone ?? '' },
  })

  const onSubmit = async (values: ProfileValues) => {
    setFormError(null)
    setSaved(false)
    try {
      const updated = await updateProfile(values)
      reset({ name: updated.name, phone: updated.phone ?? '' })
      setSaved(true)
    } catch (err) {
      setFormError(applyServerError(err, setError, ['name', 'phone']))
    }
  }

  return (
    <div className="mx-auto max-w-xl space-y-6">
      <h1 className="text-2xl font-semibold">Akun saya</h1>

      <Card className="p-5">
        <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
          <dt className="text-muted-foreground">Email</dt>
          <dd className="text-foreground">{user.email}</dd>
          <dt className="text-muted-foreground">Peran</dt>
          <dd className="text-foreground">{user.roles.map((r) => ROLE_LABELS[r] ?? r).join(', ')}</dd>
          <dt className="text-muted-foreground">Bergabung</dt>
          <dd className="text-foreground">{formatMonthYear(user.createdAt)}</dd>
        </dl>
      </Card>

      <Card className="block p-5">
        <h2 className="mb-4 text-lg font-semibold">Profil</h2>
        <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
          <FormAlert message={formError} />
          {saved && !isDirty && (
            <p role="status" className="rounded-lg border border-primary/30 bg-primary/5 px-3 py-2 text-sm text-primary">
              Profil tersimpan.
            </p>
          )}
          <TextField label="Nama" autoComplete="name" registration={register('name')} error={errors.name?.message} />
          <TextField
            label="Nomor telepon"
            type="tel"
            autoComplete="tel"
            hint="Opsional. Kosongkan untuk menghapus."
            registration={register('phone')}
            error={errors.phone?.message}
          />
          <div className="max-w-40">
            <SubmitButton busy={isSubmitting}>Simpan</SubmitButton>
          </div>
        </form>
      </Card>
    </div>
  )
}
