import { useState } from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle, Info, Minus, Plus, Store, Trash2 } from 'lucide-react'
import { ProductImage } from '../components/product/ProductImage'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { messageFor } from '../features/auth/serverErrors'
import { useCart } from '../features/cart/cartContext'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatRupiah } from '../lib/format'
import { MAX_QUANTITY_PER_LINE, type CartLine, type CartStoreGroup } from '../types/cart'

/** /cart (behind RequireAuth). All amounts come from the server. */
export function CartPage() {
  usePageTitle('Keranjang')
  const { status, cart, error, reload } = useCart()

  if (status === 'loading' || status === 'anonymous') {
    return <div className="h-64 animate-pulse rounded-xl bg-slate-200" role="status" aria-label="Memuat keranjang" />
  }
  if (status === 'error' && error) {
    return <ErrorState error={error} onRetry={reload} title="Gagal memuat keranjang" />
  }
  if (!cart || cart.stores.length === 0) {
    return (
      <EmptyState
        title="Keranjangmu masih kosong"
        description="Cari produk yang kamu suka, lalu tekan Tambah ke keranjang."
        action={
          <Link to="/products" className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700">
            Mulai belanja
          </Link>
        }
      />
    )
  }

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_20rem] lg:items-start">
      <div className="space-y-4">
        <h1 className="text-2xl font-semibold text-slate-900">Keranjang</h1>
        {cart.stores.map((group) => (
          <StoreSection key={group.store.id} group={group} />
        ))}
      </div>

      <aside className="space-y-3 rounded-xl border border-slate-200 bg-white p-5 lg:sticky lg:top-32" aria-label="Ringkasan belanja">
        <h2 className="text-lg font-semibold text-slate-900">Ringkasan belanja</h2>
        <dl className="flex justify-between text-sm text-slate-600">
          <dt>Jumlah barang</dt>
          <dd>{cart.itemCount}</dd>
        </dl>
        <dl className="flex justify-between border-t border-slate-100 pt-3 text-base font-semibold text-slate-900">
          <dt>Subtotal</dt>
          <dd>{formatRupiah(cart.subtotal)}</dd>
        </dl>
        {cart.hasIssues && (
          <p className="flex gap-2 rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-900">
            <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
            Beberapa barang tidak bisa dibeli saat ini dan tidak dihitung di subtotal.
          </p>
        )}
        <button
          type="button"
          disabled
          className="w-full rounded-md bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white opacity-50"
        >
          Lanjut ke checkout
        </button>
        <p className="flex gap-2 text-xs text-slate-500">
          <Info className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
          Checkout tersedia di tahap berikutnya. Harga dan stok dicek ulang saat checkout.
        </p>
      </aside>
    </div>
  )
}

function StoreSection({ group }: { group: CartStoreGroup }) {
  return (
    <section className="rounded-xl border border-slate-200 bg-white" aria-labelledby={`store-${group.store.id}`}>
      <header className="flex items-center justify-between border-b border-slate-100 px-5 py-3">
        <Link
          id={`store-${group.store.id}`}
          to={`/stores/${group.store.slug}`}
          className="inline-flex items-center gap-2 font-medium text-slate-900 hover:text-emerald-700"
        >
          <Store className="size-4" aria-hidden="true" /> {group.store.name}
          <span className="text-sm font-normal text-slate-500">· {group.store.location}</span>
        </Link>
        <span className="text-sm text-slate-600">{formatRupiah(group.subtotal)}</span>
      </header>
      <ul className="divide-y divide-slate-100">
        {group.items.map((line) => (
          <li key={line.id}>
            <LineRow line={line} storeSlug={group.store.slug} />
          </li>
        ))}
      </ul>
    </section>
  )
}

const ISSUE_TEXT = {
  unavailable: () => 'Produk ini sudah tidak dijual. Hapus dari keranjang.',
  out_of_stock: () => 'Stok habis.',
  insufficient_stock: (available: number) => `Stok tersisa ${available}. Kurangi jumlahnya.`,
} as const

function LineRow({ line, storeSlug }: { line: CartLine; storeSlug: string }) {
  const { update, remove } = useCart()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = async (action: () => Promise<unknown>) => {
    setBusy(true)
    setError(null)
    try {
      await action()
    } catch (err) {
      setError(messageFor(err))
    } finally {
      setBusy(false)
    }
  }

  const maxQuantity = Math.min(line.availableQuantity, MAX_QUANTITY_PER_LINE)
  const canChange = line.issue !== 'unavailable' && line.issue !== 'out_of_stock'
  const href = `/stores/${storeSlug}/products/${line.productSlug}`

  return (
    <div className="flex gap-4 px-5 py-4">
      <Link to={href} className="w-20 shrink-0">
        <ProductImage
          id={line.productId}
          name={line.productName}
          image={line.image ?? undefined}
          className={`rounded-md ${line.issue ? 'opacity-50' : ''}`}
        />
      </Link>

      <div className="min-w-0 flex-1 space-y-1">
        <Link to={href} className="line-clamp-2 text-sm font-medium text-slate-900 hover:text-emerald-700">
          {line.productName}
        </Link>
        <p className="text-xs text-slate-500">Varian: {line.variantName}</p>
        <p className="text-sm font-semibold text-slate-900">{formatRupiah(line.unitPrice)}</p>

        {line.issue && (
          <p role="status" className="flex items-start gap-1 text-sm text-amber-800">
            <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
            {ISSUE_TEXT[line.issue](line.availableQuantity)}
            {line.issue === 'insufficient_stock' && (
              <button
                type="button"
                disabled={busy}
                onClick={() => run(() => update(line.id, line.availableQuantity))}
                className="ml-1 font-medium underline"
              >
                Ubah ke {line.availableQuantity}
              </button>
            )}
          </p>
        )}
        {error && (
          <p role="alert" className="text-sm text-red-600">
            {error}
          </p>
        )}
      </div>

      <div className="flex flex-col items-end justify-between gap-2">
        <span className="text-sm font-semibold text-slate-900">{formatRupiah(line.lineTotal)}</span>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => run(() => remove(line.id))}
            disabled={busy}
            aria-label={`Hapus ${line.productName} dari keranjang`}
            className="rounded-md p-2 text-slate-500 hover:bg-red-50 hover:text-red-600 disabled:opacity-40"
          >
            <Trash2 className="size-4" aria-hidden="true" />
          </button>
          <div role="group" aria-label={`Jumlah ${line.productName}`} className="inline-flex items-center rounded-md border border-slate-300">
            <button
              type="button"
              aria-label="Kurangi jumlah"
              disabled={busy || !canChange || line.quantity <= 1}
              onClick={() => run(() => update(line.id, line.quantity - 1))}
              className="p-1.5 text-slate-700 disabled:opacity-40"
            >
              <Minus className="size-4" aria-hidden="true" />
            </button>
            <output aria-live="polite" className="min-w-8 text-center text-sm">
              {line.quantity}
            </output>
            <button
              type="button"
              aria-label="Tambah jumlah"
              disabled={busy || !canChange || line.quantity >= maxQuantity}
              onClick={() => run(() => update(line.id, line.quantity + 1))}
              className="p-1.5 text-slate-700 disabled:opacity-40"
            >
              <Plus className="size-4" aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
