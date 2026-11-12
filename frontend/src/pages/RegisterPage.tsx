import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { FormAlert, TextField } from '../components/shared/TextField'
import { useAuth } from '../features/auth/authContext'
import { safeNextPath } from '../features/auth/redirect'
import { registerSchema, type RegisterValues } from '../features/auth/schemas'
import { applyServerError } from '../features/auth/serverErrors'
import { AuthCard, SubmitButton } from '../features/auth/AuthCard'
import { usePageTitle } from '../hooks/usePageTitle'

export function RegisterPage() {
  usePageTitle('Daftar')
  const { status, register: registerAccount } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const next = safeNextPath(params.get('next'))
  const [formError, setFormError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<RegisterValues>({ resolver: zodResolver(registerSchema) })

  if (status === 'authenticated') return <Navigate to={next} replace />

  const onSubmit = async (values: RegisterValues) => {
    setFormError(null)
    try {
      await registerAccount(values) // the server logs the new account in
      navigate(next, { replace: true })
    } catch (err) {
      setFormError(applyServerError(err, setError, ['name', 'email', 'password']))
    }
  }

  return (
    <AuthCard title="Buat akun">
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
        <FormAlert message={formError} />
        <TextField label="Nama" autoComplete="name" registration={register('name')} error={errors.name?.message} />
        <TextField label="Email" type="email" autoComplete="email" registration={register('email')} error={errors.email?.message} />
        <TextField
          label="Kata sandi"
          type="password"
          autoComplete="new-password"
          hint="Minimal 8 karakter."
          registration={register('password')}
          error={errors.password?.message}
        />
        <SubmitButton busy={isSubmitting}>Daftar</SubmitButton>
      </form>
      <p className="mt-6 text-center text-sm text-slate-600">
        Sudah punya akun?{' '}
        <Link to={`/login${next !== '/' ? `?next=${encodeURIComponent(next)}` : ''}`} className="font-medium text-emerald-700 hover:underline">
          Masuk
        </Link>
      </p>
    </AuthCard>
  )
}
