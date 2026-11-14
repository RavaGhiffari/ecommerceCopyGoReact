import { useId, type InputHTMLAttributes } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'
import { AlertCircle } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { FieldMessage } from './FieldMessage'

interface TextFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'name'> {
  label: string
  registration: UseFormRegisterReturn
  error?: string
  hint?: string
}

/**
 * Label + shadcn Input + error/hint, wired for screen readers.
 * `registration` is react-hook-form's register(...) result; spreading it
 * passes the ref, so RHF reads the real <input> element (React 19 passes
 * ref as a normal prop to function components).
 */
export function TextField({ label, registration, error, hint, ...inputProps }: TextFieldProps) {
  const id = useId()
  const describedBy = error ? `${id}-error` : hint ? `${id}-hint` : undefined

  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className="h-9"
        {...inputProps}
        {...registration}
      />
      <FieldMessage id={id} error={error} hint={hint} />
    </div>
  )
}

/** Form-level error (e.g. "Email atau kata sandi salah."); role="alert" is set by Alert. */
export function FormAlert({ message }: { message: string | null }) {
  if (!message) return null
  return (
    <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
      <AlertCircle aria-hidden="true" />
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  )
}
