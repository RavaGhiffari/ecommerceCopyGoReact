import { useState } from 'react'
import { Controller, useFieldArray, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate } from 'react-router-dom'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { TextAreaField } from '../components/shared/TextAreaField'
import { FormAlert, TextField } from '../components/shared/TextField'
import { SubmitButton } from '../features/auth/AuthCard'
import { applyServerError } from '../features/auth/serverErrors'
import { CategorySelect } from '../features/seller/CategorySelect'
import { newProductSchema, parseImageLines, type NewProductValues } from '../features/seller/schemas'
import { usePageTitle } from '../hooks/usePageTitle'
import { createProduct } from '../services/seller'

const emptyVariant = { name: '', sku: '', price: '', stock: '0' }

/** /seller/products/new. A product is created together with its variants and stock. */
export function SellerNewProductPage() {
  usePageTitle('Tambah produk')
  const navigate = useNavigate()
  const [formError, setFormError] = useState<string | null>(null)

  const {
    register,
    control,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<NewProductValues>({
    resolver: zodResolver(newProductSchema),
    defaultValues: { name: '', categoryId: '', description: '', images: '', variants: [emptyVariant] },
  })
  const { fields, append, remove } = useFieldArray({ control, name: 'variants' })

  const onSubmit = async (v: NewProductValues) => {
    setFormError(null)
    try {
      const created = await createProduct({
        name: v.name,
        categoryId: v.categoryId,
        description: v.description,
        images: parseImageLines(v.images).map((url) => ({ url, alt: v.name })),
        variants: v.variants.map((x) => ({ name: x.name, sku: x.sku, price: x.price, stock: Number(x.stock) })),
      })
      navigate(`/seller/products/${created.id}`, { replace: true })
    } catch (err) {
      setFormError(applyServerError(err, setError, ['name', 'categoryId', 'description', 'images', 'variants']))
    }
  }

  const variantsError = errors.variants?.message ?? errors.variants?.root?.message

  return (
    <div className="mx-auto max-w-3xl">
      <Link to="/seller" className="text-sm text-muted-foreground hover:text-primary">
        ← Kembali ke toko
      </Link>
      <h1 className="mb-6 mt-2 text-2xl font-semibold">Tambah produk</h1>

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-6">
        <FormAlert message={formError} />

        <Card className="space-y-4 p-5">
          <TextField label="Nama produk" registration={register('name')} error={errors.name?.message} />
          <Controller
            name="categoryId"
            control={control}
            render={({ field }) => (
              <CategorySelect
                value={field.value}
                onChange={field.onChange}
                onBlur={field.onBlur}
                ref={field.ref}
                error={errors.categoryId?.message}
              />
            )}
          />
          <TextAreaField label="Deskripsi" rows={4} registration={register('description')} error={errors.description?.message} />
          <TextAreaField
            label="URL gambar (opsional)"
            rows={3}
            placeholder="https://contoh.com/foto-1.jpg"
            hint="Satu URL per baris, maksimal 8. Tanpa gambar, katalog menampilkan placeholder."
            registration={register('images')}
            error={errors.images?.message}
          />
        </Card>

        <Card className="gap-0 p-5">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-lg font-semibold">Varian</h2>
            <Button type="button" variant="ghost" className="text-primary" onClick={() => append(emptyVariant)} disabled={fields.length >= 20}>
              <Plus aria-hidden="true" /> Tambah varian
            </Button>
          </div>
          <p className="mb-4 text-sm text-muted-foreground">Setiap varian punya harga dan stok sendiri, misalnya ukuran atau warna.</p>
          {variantsError && <p className="mb-3 text-sm text-destructive" role="alert">{variantsError}</p>}

          <div className="space-y-4">
            {fields.map((field, i) => (
              <fieldset key={field.id} className="grid gap-3 rounded-lg border p-3 sm:grid-cols-[2fr_1.5fr_1.5fr_1fr_auto] sm:items-start">
                <legend className="sr-only">Varian {i + 1}</legend>
                <TextField label="Nama" placeholder="Hitam / M" registration={register(`variants.${i}.name`)} error={errors.variants?.[i]?.name?.message} />
                <TextField label="SKU" placeholder="KMJ-01" registration={register(`variants.${i}.sku`)} error={errors.variants?.[i]?.sku?.message} />
                <TextField label="Harga (Rp)" inputMode="decimal" registration={register(`variants.${i}.price`)} error={errors.variants?.[i]?.price?.message} />
                <TextField label="Stok" inputMode="numeric" registration={register(`variants.${i}.stock`)} error={errors.variants?.[i]?.stock?.message} />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-lg"
                  onClick={() => remove(i)}
                  disabled={fields.length === 1}
                  aria-label={`Hapus varian ${i + 1}`}
                  className="self-end text-muted-foreground hover:bg-destructive/10 hover:text-destructive sm:mt-6"
                >
                  <Trash2 aria-hidden="true" />
                </Button>
              </fieldset>
            ))}
          </div>
        </Card>

        <div className="max-w-48">
          <SubmitButton busy={isSubmitting}>Simpan produk</SubmitButton>
        </div>
      </form>
    </div>
  )
}
