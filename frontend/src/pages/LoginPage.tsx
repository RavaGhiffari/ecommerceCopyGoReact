import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { FormAlert, TextField } from '../components/shared/TextField'
import { AuthCard, SubmitButton } from '../features/auth/AuthCard'
import { useAuth } from '../features/auth/authContext'
import { safeNextPath } from '../features/auth/redirect'
import { loginSchema, type LoginValues } from '../features/auth/schemas'
import { applyServerError } from '../features/auth/serverErrors'
import { usePageTitle } from '../hooks/usePageTitle'

export function LoginPage() {
  usePageTitle('Masuk')
  const { status, login } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const next = safeNextPath(params.get('next'))
  const [formError, setFormError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<LoginValues>({ resolver: zodResolver(loginSchema) })

  if (status === 'authenticated') return <Navigate to={next} replace />

  const onSubmit = async (values: LoginValues) => {
    setFormError(null)
    try {
      await login(values)
      navigate(next, { replace: true })
    } catch (err) {
      setFormError(applyServerError(err, setError, ['email', 'password']))
    }
  }

  return (
    <AuthCard title="Masuk ke NiagaHeula">
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
        <FormAlert message={formError} />
        <TextField label="Email" type="email" autoComplete="email" registration={register('email')} error={errors.email?.message} />
        <TextField
          label="Kata sandi"
          type="password"
          autoComplete="current-password"
          registration={register('password')}
          error={errors.password?.message}
        />
        <SubmitButton busy={isSubmitting}>Masuk</SubmitButton>
      </form>
      <p className="mt-6 text-center text-sm text-muted-foreground">
        Belum punya akun?{' '}
        <Link to={`/register${next !== '/' ? `?next=${encodeURIComponent(next)}` : ''}`} className="font-medium text-primary hover:underline">
          Daftar
        </Link>
      </p>
    </AuthCard>
  )
}
