import { useId, type Ref } from 'react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { FieldMessage } from '../../components/shared/FieldMessage'
import type { Category } from '../../types/marketplace'
import { useCategories } from '../catalog/hooks'

interface CategorySelectProps {
  /** Selected category id; '' shows the placeholder. */
  value: string
  onChange: (categoryId: string) => void
  onBlur?: () => void
  /** react-hook-form's field.ref, so a validation error can focus the trigger. */
  ref?: Ref<HTMLButtonElement>
  error?: string
}

/**
 * Category picker for products (Radix Select via shadcn). Products go on a
 * LEAF category: subcategories are listed under their parent's heading, and a
 * parent that has subcategories cannot be picked itself. The API enforces the
 * same rule.
 *
 * Controlled component: use it through react-hook-form's <Controller>, not
 * register(). Radix Select is a button + listbox, not a native <select>, so
 * RHF cannot read its value from the DOM.
 *
 * The select is only mounted once the categories have loaded, so the trigger
 * never shows an id without its name. (With the old native <select> this also
 * prevented RHF from losing the default value; see PROJECT_STATE §11.)
 */
export function CategorySelect({ value, onChange, onBlur, ref, error }: CategorySelectProps) {
  const id = useId()
  const { state, reload } = useCategories()

  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>Kategori</Label>
      {state.status === 'success' ? (
        <Select
          value={value}
          onValueChange={onChange}
          onOpenChange={(open) => {
            if (!open) onBlur?.()
          }}
        >
          <SelectTrigger
            id={id}
            ref={ref}
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? `${id}-error` : undefined}
            className="h-9 w-full"
          >
            <SelectValue placeholder="Pilih kategori" />
          </SelectTrigger>
          <SelectContent position="popper">
            <CategoryItems categories={state.data} />
          </SelectContent>
        </Select>
      ) : (
        <Select disabled>
          <SelectTrigger id={id} className="h-9 w-full">
            <SelectValue placeholder={state.status === 'loading' ? 'Memuat kategori…' : 'Kategori tidak tersedia'} />
          </SelectTrigger>
        </Select>
      )}
      {state.status === 'error' && (
        <p className="text-sm text-destructive">
          Gagal memuat kategori.{' '}
          <Button type="button" variant="link" className="h-auto p-0 text-destructive underline" onClick={reload}>
            Coba lagi
          </Button>
        </p>
      )}
      <FieldMessage id={id} error={error} />
    </div>
  )
}

function CategoryItems({ categories }: { categories: Category[] }) {
  return categories
    .filter((c) => c.parentId === null)
    .map((parent) => {
      const children = categories.filter((c) => c.parentId === parent.id)
      return children.length > 0 ? (
        <SelectGroup key={parent.id}>
          <SelectLabel>{parent.name}</SelectLabel>
          {children.map((c) => (
            <SelectItem key={c.id} value={c.id}>
              {c.name}
            </SelectItem>
          ))}
        </SelectGroup>
      ) : (
        <SelectItem key={parent.id} value={parent.id}>
          {parent.name}
        </SelectItem>
      )
    })
}
