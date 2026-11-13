import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useParams } from 'react-router-dom'
import { ExternalLink } from 'lucide-react'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { TextAreaField } from '../components/shared/TextAreaField'
import { FormAlert, TextField } from '../components/shared/TextField'
import { SubmitButton } from '../features/auth/AuthCard'
import { applyServerError } from '../features/auth/serverErrors'
import { CategorySelect } from '../features/seller/CategorySelect'
import { useMyProduct, useMyStore } from '../features/seller/hooks'
import {
  parseImageLines,
  priceField,
  productSchema,
  stockField,
  variantSchema,
  type ProductValues,
  type VariantValues,
} from '../features/seller/schemas'
import { usePageTitle } from '../hooks/usePageTitle'
import { ApiError } from '../lib/api'
import { formatRupiah } from '../lib/format'
import { addVariant, archiveProduct, setStock, updateProduct, updateVariant } from '../services/seller'
import type { ProductVariant } from '../types/marketplace'
import type { SellerProduct } from '../types/seller'

/** /seller/products/:id. Every change is saved separately and the product is reloaded. */
export function SellerEditProductPage() {
  const { id = '' } = useParams()
  const product = useMyProduct(id)
  const store = useMyStore()
  usePageTitle(product.state.status === 'success' ? product.state.data.name : 'Produk')

  if (product.state.status === 'loading') {
    return <div className="mx-auto h-96 max-w-3xl animate-pulse rounded-xl bg-slate-200" role="status" aria-label="Memuat produk" />
  }
  if (product.state.status === 'error') {
    return product.state.error.status === 404 ? (
      <EmptyState
        title="Produk tidak ditemukan"
        description="Produk ini tidak ada di toko kamu."
        action={<Link to="/seller" className="text-sm font-medium text-emerald-700 hover:underline">Kembali ke toko</Link>}
      />
    ) : (
      <ErrorState error={product.state.error} onRetry={product.reload} title="Gagal memuat produk" />
    )
  }

  const p = product.state.data
  const storeSlug = store.state.status === 'success' ? store.state.data.slug : null

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div>
        <Link to="/seller" className="text-sm text-slate-500 hover:text-emerald-700">← Kembali ke toko</Link>
        <div className="mt-2 flex flex-wrap items-center justify-between gap-3">
          <h1 className="text-2xl font-semibold text-slate-900">{p.name}</h1>
          <StatusControl product={p} onChanged={product.reload} />
        </div>
        {p.status === 'active' && storeSlug && (
          <Link to={`/stores/${storeSlug}/products/${p.slug}`} className="mt-1 inline-flex items-center gap-1 text-sm text-emerald-700 hover:underline">
            Lihat di katalog <ExternalLink className="size-3.5" aria-hidden="true" />
          </Link>
        )}
      </div>

      {/* key: remount the form with fresh defaults after the product changes */}
      <DetailsForm key={p.updatedAt} product={p} onSaved={product.reload} />

      <section className="rounded-xl border border-slate-200 bg-white p-5">
        <h2 className="mb-1 text-lg font-semibold text-slate-900">Varian, harga & stok</h2>
        <p className="mb-4 text-sm text-slate-600">
          Perubahan stok tercatat di riwayat stok. Mengubah harga tidak mengubah pesanan yang sudah ada.
        </p>
        <div className="divide-y divide-slate-100">
          {p.variants.map((v) => (
            <VariantRow key={`${v.id}:${v.price}:${v.stock}`} productId={p.id} variant={v} onSaved={product.reload} />
          ))}
        </div>
        <AddVariantForm productId={p.id} disabled={p.variants.length >= 20} onAdded={product.reload} />
      </section>
    </div>
  )
}

function StatusControl({ product, onChanged }: { product: SellerProduct; onChanged: () => void }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const archived = product.status === 'archived'

  const toggle = async () => {
    setBusy(true)
    setError(null)
    try {
      if (archived) await updateProduct(product.id, { status: 'active' })
      else await archiveProduct(product.id)
      onChanged()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Gagal mengubah status')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex items-center gap-3">
      <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${archived ? 'bg-slate-200 text-slate-700' : 'bg-emerald-100 text-emerald-800'}`}>
        {archived ? 'Diarsipkan' : 'Aktif'}
      </span>
      <button
        type="button"
        onClick={toggle}
        disabled={busy}
        className="rounded-md border border-slate-300 px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50 disabled:opacity-50"
      >
        {archived ? 'Aktifkan lagi' : 'Arsipkan'}
      </button>
      {error && <span className="text-sm text-red-600" role="alert">{error}</span>}
    </div>
  )
}

function DetailsForm({ product, onSaved }: { product: SellerProduct; onSaved: () => void }) {
  const [formError, setFormError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<ProductValues>({
    resolver: zodResolver(productSchema),
    defaultValues: {
      name: product.name,
      categoryId: product.categoryId,
      description: product.description,
      images: product.images.map((i) => i.url).join('\n'),
    },
  })

  const onSubmit = async (v: ProductValues) => {
    setFormError(null)
    try {
      await updateProduct(product.id, {
        name: v.name,
        categoryId: v.categoryId,
        description: v.description,
        images: parseImageLines(v.images).map((url) => ({ url, alt: v.name })),
      })
      onSaved()
    } catch (err) {
      setFormError(applyServerError(err, setError, ['name', 'categoryId', 'description', 'images']))
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4 rounded-xl border border-slate-200 bg-white p-5">
      <h2 className="text-lg font-semibold text-slate-900">Detail produk</h2>
      <FormAlert message={formError} />
      <TextField label="Nama produk" registration={register('name')} error={errors.name?.message} />
      <CategorySelect registration={register('categoryId')} error={errors.categoryId?.message} />
      <TextAreaField label="Deskripsi" rows={4} registration={register('description')} error={errors.description?.message} />
      <TextAreaField label="URL gambar" rows={3} hint="Satu URL per baris, maksimal 8." registration={register('images')} error={errors.images?.message} />
      <div className="max-w-40">
        <SubmitButton busy={isSubmitting}>{isDirty ? 'Simpan' : 'Tersimpan'}</SubmitButton>
      </div>
    </form>
  )
}

function apiMessage(err: unknown): string {
  if (!(err instanceof ApiError)) return 'Terjadi kesalahan'
  const field = Object.values(err.fieldErrors())[0]
  return field ? `${err.message}: ${field}` : err.message
}

/** One variant: edit its price, set its stock. Each saves on its own. */
function VariantRow({ productId, variant, onSaved }: { productId: string; variant: ProductVariant; onSaved: () => void }) {
  const [price, setPrice] = useState(String(Number(variant.price)))
  const [stock, setStockValue] = useState(String(variant.stock))
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState<{ kind: 'error' | 'ok'; text: string } | null>(null)

  const run = async (action: () => Promise<unknown>, ok: string) => {
    setBusy(true)
    setMessage(null)
    try {
      await action()
      setMessage({ kind: 'ok', text: ok })
      onSaved()
    } catch (err) {
      setMessage({ kind: 'error', text: apiMessage(err) })
    } finally {
      setBusy(false)
    }
  }

  const savePrice = () => {
    const parsed = priceField.safeParse(price)
    if (!parsed.success) return setMessage({ kind: 'error', text: parsed.error.issues[0].message })
    void run(() => updateVariant(productId, variant.id, { price: parsed.data }), 'Harga disimpan')
  }
  const saveStock = () => {
    const parsed = stockField.safeParse(stock)
    if (!parsed.success) return setMessage({ kind: 'error', text: parsed.error.issues[0].message })
    void run(() => setStock(productId, variant.id, Number(parsed.data), 'penyesuaian dari dasbor'), 'Stok disimpan')
  }

  return (
    <div className="grid gap-3 py-3 sm:grid-cols-[1.5fr_2fr_2fr] sm:items-end">
      <div>
        <p className="font-medium text-slate-900">{variant.name}</p>
        <p className="text-xs text-slate-500">SKU {variant.sku} · {formatRupiah(variant.price)} · stok {variant.stock}</p>
      </div>
      <label className="text-sm text-slate-700">
        Harga (Rp)
        <span className="mt-1 flex gap-2">
          <input value={price} onChange={(e) => setPrice(e.target.value)} inputMode="decimal" className="w-full min-w-0 rounded-md border border-slate-300 px-2 py-1.5 text-sm" />
          <button type="button" onClick={savePrice} disabled={busy} className="rounded-md border border-slate-300 px-2 text-sm hover:bg-slate-50 disabled:opacity-50">Simpan</button>
        </span>
      </label>
      <label className="text-sm text-slate-700">
        Stok
        <span className="mt-1 flex gap-2">
          <input value={stock} onChange={(e) => setStockValue(e.target.value)} inputMode="numeric" className="w-full min-w-0 rounded-md border border-slate-300 px-2 py-1.5 text-sm" />
          <button type="button" onClick={saveStock} disabled={busy} className="rounded-md border border-slate-300 px-2 text-sm hover:bg-slate-50 disabled:opacity-50">Simpan</button>
        </span>
      </label>
      {message && (
        <p role={message.kind === 'error' ? 'alert' : 'status'} className={`text-sm sm:col-span-3 ${message.kind === 'error' ? 'text-red-600' : 'text-emerald-700'}`}>
          {message.text}
        </p>
      )}
    </div>
  )
}

function AddVariantForm({ productId, disabled, onAdded }: { productId: string; disabled: boolean; onAdded: () => void }) {
  const [formError, setFormError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<VariantValues>({ resolver: zodResolver(variantSchema), defaultValues: { name: '', sku: '', price: '', stock: '0' } })

  const onSubmit = async (v: VariantValues) => {
    setFormError(null)
    try {
      await addVariant(productId, { name: v.name, sku: v.sku, price: v.price, stock: Number(v.stock) })
      reset()
      onAdded()
    } catch (err) {
      setFormError(applyServerError(err, setError, ['name', 'sku', 'price', 'stock']))
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="mt-4 border-t border-slate-200 pt-4">
      <h3 className="mb-3 text-sm font-semibold text-slate-900">Tambah varian</h3>
      <FormAlert message={formError} />
      <div className="grid gap-3 sm:grid-cols-[2fr_1.5fr_1.5fr_1fr_auto] sm:items-start">
        <TextField label="Nama" registration={register('name')} error={errors.name?.message} />
        <TextField label="SKU" registration={register('sku')} error={errors.sku?.message} />
        <TextField label="Harga (Rp)" inputMode="decimal" registration={register('price')} error={errors.price?.message} />
        <TextField label="Stok" inputMode="numeric" registration={register('stock')} error={errors.stock?.message} />
        <button
          type="submit"
          disabled={disabled || isSubmitting}
          className="self-end rounded-md bg-emerald-600 px-3 py-2 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
        >
          Tambah
        </button>
      </div>
    </form>
  )
}
