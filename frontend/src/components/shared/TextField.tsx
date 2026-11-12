import { useId, type InputHTMLAttributes } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'

interface TextFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'name'> {
  label: string
  registration: UseFormRegisterReturn
  error?: string
  hint?: string
}

/** Label + input + error message, wired for screen readers. */
export function TextField({ label, registration, error, hint, ...inputProps }: TextFieldProps) {
  const id = useId()
  const describedBy = error ? `${id}-error` : hint ? `${id}-hint` : undefined

  return (
    <div>
      <label htmlFor={id} className="mb-1 block text-sm font-medium text-slate-700">
        {label}
      </label>
      <input
        id={id}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className={`w-full rounded-md border px-3 py-2 text-sm focus:outline-none focus-visible:ring-2 ${
          error
            ? 'border-red-400 focus-visible:ring-red-500'
            : 'border-slate-300 focus-visible:ring-emerald-600'
        }`}
        {...inputProps}
        {...registration}
      />
      {error ? (
        <p id={`${id}-error`} className="mt-1 text-sm text-red-600">
          {error}
        </p>
      ) : (
        hint && (
          <p id={`${id}-hint`} className="mt-1 text-xs text-slate-500">
            {hint}
          </p>
        )
      )}
    </div>
  )
}

export function FormAlert({ message }: { message: string | null }) {
  if (!message) return null
  return (
    <p role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
      {message}
    </p>
  )
}
