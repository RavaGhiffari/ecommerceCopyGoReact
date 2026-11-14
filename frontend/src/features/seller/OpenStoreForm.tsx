import { Card } from '@/components/ui/card'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { TextAreaField } from '../../components/shared/TextAreaField'
import { FormAlert, TextField } from '../../components/shared/TextField'
import { openStore } from '../../services/seller'
import { SubmitButton } from '../auth/AuthCard'
import { useAuth } from '../auth/authContext'
import { applyServerError } from '../auth/serverErrors'
import { openStoreSchema, type OpenStoreValues } from './schemas'

/**
 * Opening a store is how a customer becomes a seller. The server creates the
 * store and grants the seller role in one transaction; afterwards we reload
 * /me so the UI learns about the new role.
 */
export function OpenStoreForm() {
  const { refresh } = useAuth()
  const [formError, setFormError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<OpenStoreValues>({ resolver: zodResolver(openStoreSchema), defaultValues: { description: '' } })

  const onSubmit = async (values: OpenStoreValues) => {
    setFormError(null)
    try {
      await openStore(values)
      refresh()
    } catch (err) {
      setFormError(applyServerError(err, setError, ['name', 'location', 'description']))
    }
  }

  return (
    <Card className="mx-auto block max-w-lg p-6 sm:p-8">
      <h1 className="text-2xl font-semibold">Buka toko</h1>
      <p className="mt-1 text-sm text-muted-foreground">
        Satu akun bisa memiliki satu toko. Alamat toko dibuat dari namanya dan tidak bisa diubah nanti.
      </p>
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="mt-6 space-y-4">
        <FormAlert message={formError} />
        <TextField label="Nama toko" registration={register('name')} error={errors.name?.message} />
        <TextField label="Lokasi" placeholder="Contoh: Bandung" registration={register('location')} error={errors.location?.message} />
        <TextAreaField label="Deskripsi (opsional)" rows={3} registration={register('description')} error={errors.description?.message} />
        <SubmitButton busy={isSubmitting}>Buka toko</SubmitButton>
      </form>
    </Card>
  )
}
