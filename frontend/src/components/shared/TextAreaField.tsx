import { useId, type TextareaHTMLAttributes } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'

interface TextAreaFieldProps extends Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'name'> {
  label: string
  registration: UseFormRegisterReturn
  error?: string
  hint?: string
}

export function TextAreaField({ label, registration, error, hint, rows = 4, ...props }: TextAreaFieldProps) {
  const id = useId()
  const describedBy = error ? `${id}-error` : hint ? `${id}-hint` : undefined
  return (
    <div>
      <label htmlFor={id} className="mb-1 block text-sm font-medium text-slate-700">
        {label}
      </label>
      <textarea
        id={id}
        rows={rows}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className={`w-full rounded-md border px-3 py-2 text-sm focus:outline-none focus-visible:ring-2 ${
          error ? 'border-red-400 focus-visible:ring-red-500' : 'border-slate-300 focus-visible:ring-emerald-600'
        }`}
        {...props}
        {...registration}
      />
      {error ? (
        <p id={`${id}-error`} className="mt-1 text-sm text-red-600">{error}</p>
      ) : (
        hint && <p id={`${id}-hint`} className="mt-1 text-xs text-slate-500">{hint}</p>
      )}
    </div>
  )
}
