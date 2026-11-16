import { useId, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { FieldMessage } from '../../components/shared/FieldMessage'
import { FormAlert, TextField } from '../../components/shared/TextField'
import { createCategory, replaceCategory } from '../../services/admin'
import type { Category } from '../../types/marketplace'
import { applyServerError } from '../auth/serverErrors'
import { CATEGORY_FIELDS, NO_PARENT, categorySchema, type CategoryValues } from './categorySchema'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** The category to edit; undefined = create. */
  category?: Category
  /** Every category, to offer parents and to know whether `category` has children. */
  all: Category[]
  onSaved: (saved: Category, created: boolean) => void
}

/** Create / edit a category through the admin API (each save writes an audit row). */
export function CategoryFormDialog({ open, onOpenChange, category, all, onSaved }: Props) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{category ? `Ubah kategori “${category.name}”` : 'Tambah kategori'}</DialogTitle>
          <DialogDescription>Kategori hanya dua tingkat: kategori utama dan subkategori.</DialogDescription>
        </DialogHeader>
        <CategoryForm key={category?.id ?? 'new'} category={category} all={all} onSaved={onSaved} />
      </DialogContent>
    </Dialog>
  )
}

function CategoryForm({ category, all, onSaved }: Pick<Props, 'category' | 'all' | 'onSaved'>) {
  const parentFieldId = useId()
  const [formError, setFormError] = useState<string | null>(null)
  const hasChildren = category ? all.some((c) => c.parentId === category.id) : false
  // A parent must be top-level and cannot be the category itself.
  const parents = all.filter((c) => c.parentId === null && c.id !== category?.id)

  const {
    register,
    control,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<CategoryValues>({
    resolver: zodResolver(categorySchema),
    defaultValues: {
      name: category?.name ?? '',
      slug: category?.slug ?? '',
      parentId: category?.parentId ?? NO_PARENT,
      sortOrder: String(category?.sortOrder ?? 0),
    },
  })

  const onSubmit = async (v: CategoryValues) => {
    setFormError(null)
    const input = {
      name: v.name,
      slug: v.slug,
      parentId: v.parentId === NO_PARENT ? null : v.parentId,
      sortOrder: Number(v.sortOrder),
    }
    try {
      const saved = category ? await replaceCategory(category.id, input) : await createCategory(input)
      onSaved(saved, !category)
    } catch (err) {
      setFormError(applyServerError(err, setError, CATEGORY_FIELDS))
    }
  }

  return (
    <form onSubmit={(e) => void handleSubmit(onSubmit)(e)} noValidate className="space-y-4">
      <TextField label="Nama" registration={register('name')} error={errors.name?.message} maxLength={100} />
      <TextField
        label="Slug"
        registration={register('slug')}
        error={errors.slug?.message}
        hint={
          category
            ? 'Mengubah slug mengubah alamat halaman kategori (/categories/slug).'
            : 'Kosongkan untuk dibuat otomatis dari nama.'
        }
      />

      <div className="space-y-1.5">
        <Label htmlFor={parentFieldId}>Induk</Label>
        <Controller
          control={control}
          name="parentId"
          render={({ field }) => (
            <Select value={field.value} onValueChange={field.onChange} disabled={hasChildren}>
              <SelectTrigger
                id={parentFieldId}
                ref={field.ref}
                aria-invalid={errors.parentId ? true : undefined}
                aria-describedby={
                  errors.parentId ? `${parentFieldId}-error` : hasChildren ? `${parentFieldId}-hint` : undefined
                }
                className="h-9 w-full"
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent position="popper">
                <SelectItem value={NO_PARENT}>Tidak ada (kategori utama)</SelectItem>
                {parents.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        />
        <FieldMessage
          id={parentFieldId}
          error={errors.parentId?.message}
          hint={hasChildren ? 'Kategori ini punya subkategori, jadi harus tetap kategori utama.' : undefined}
        />
      </div>

      <TextField
        label="Urutan"
        type="number"
        inputMode="numeric"
        step={1}
        registration={register('sortOrder')}
        error={errors.sortOrder?.message}
        hint="Angka kecil tampil lebih dulu."
      />

      <FormAlert message={formError} />
      <DialogFooter>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Menyimpan…' : category ? 'Simpan perubahan' : 'Tambah kategori'}
        </Button>
      </DialogFooter>
    </form>
  )
}
