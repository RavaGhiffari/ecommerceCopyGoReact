import { useId, type TextareaHTMLAttributes } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { FieldMessage } from './FieldMessage'

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
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      {/* field-sizing-fixed: keep the rows height instead of shadcn's auto-grow */}
      <Textarea
        id={id}
        rows={rows}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className="field-sizing-fixed"
        {...props}
        {...registration}
      />
      <FieldMessage id={id} error={error} hint={hint} />
    </div>
  )
}
