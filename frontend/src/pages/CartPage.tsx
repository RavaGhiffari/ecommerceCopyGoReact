import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AlertTriangle, Info, Minus, Plus, Store, Trash2 } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ItemRowSkeleton, LineSkeleton, LoadingRegion } from '../components/shared/Skeletons'
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
  const navigate = useNavigate()

  if (status === 'loading' || status === 'anonymous') {
    return <CartSkeleton />
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
          <Button size="lg" asChild>
            <Link to="/products">Mulai belanja</Link>
          </Button>
        }
      />
    )
  }

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_20rem] lg:items-start">
      <div className="space-y-4">
        <h1 className="text-2xl font-semibold">Keranjang</h1>
        {cart.stores.map((group) => (
          <StoreSection key={group.store.id} group={group} />
        ))}
      </div>

      <aside className="lg:sticky lg:top-32" aria-label="Ringkasan belanja">
       <Card className="gap-3 p-5">
        <h2 className="text-lg font-semibold">Ringkasan belanja</h2>
        <dl className="flex justify-between text-sm text-muted-foreground">
          <dt>Jumlah barang</dt>
          <dd>{cart.itemCount}</dd>
        </dl>
        <dl className="flex justify-between border-t pt-3 text-base font-semibold">
          <dt>Subtotal</dt>
          <dd>{formatRupiah(cart.subtotal)}</dd>
        </dl>
        {cart.hasIssues && (
          <Alert role="note" className="border-amber-200 bg-amber-50 text-amber-900">
            <AlertTriangle aria-hidden="true" />
            <AlertDescription className="text-amber-900">
              Beberapa barang tidak bisa dibeli saat ini dan tidak dihitung di subtotal.
            </AlertDescription>
          </Alert>
        )}
        <Button
          size="lg"
          disabled={cart.hasIssues}
          onClick={() => navigate('/checkout')}
          className="h-10 w-full font-semibold"
        >
          Lanjut ke checkout
        </Button>
        <p className="flex gap-2 text-xs text-muted-foreground">
          <Info className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
          {cart.hasIssues
            ? 'Perbaiki atau hapus barang yang bermasalah dulu. Semua isi keranjang ikut di-checkout.'
            : 'Semua isi keranjang ikut di-checkout. Harga dan stok dicek ulang saat pesanan dibuat.'}
        </p>
       </Card>
      </aside>
    </div>
  )
}

/** Same grid as the cart: two store cards with two lines each, and the summary. */
function CartSkeleton() {
  return (
    <LoadingRegion label="Memuat keranjang" className="grid gap-6 lg:grid-cols-[1fr_20rem] lg:items-start">
      <div className="space-y-4">
        <Skeleton className="h-8 w-40" />
        {[0, 1].map((i) => (
          <Card key={i} className="gap-0 py-0">
            <div className="flex items-center justify-between border-b px-5 py-3">
              <LineSkeleton className="w-48" />
              <LineSkeleton className="w-24" />
            </div>
            <div className="divide-y">
              <ItemRowSkeleton />
              <ItemRowSkeleton />
            </div>
          </Card>
        ))}
      </div>
      <Card className="gap-3 p-5">
        <Skeleton className="h-6 w-40" />
        <div className="flex justify-between">
          <LineSkeleton className="w-24" />
          <LineSkeleton className="w-8" />
        </div>
        <div className="flex justify-between border-t pt-3">
          <LineSkeleton className="w-20" />
          <LineSkeleton className="w-28" />
        </div>
        <Skeleton className="h-10 w-full" />
        <LineSkeleton className="h-3 w-full" />
      </Card>
    </LoadingRegion>
  )
}

function StoreSection({ group }: { group: CartStoreGroup }) {
  return (
    <Card className="gap-0 py-0">
     <section aria-labelledby={`store-${group.store.id}`}>
      <header className="flex items-center justify-between border-b px-5 py-3">
        <Link
          id={`store-${group.store.id}`}
          to={`/stores/${group.store.slug}`}
          className="inline-flex items-center gap-2 font-medium hover:text-primary"
        >
          <Store className="size-4" aria-hidden="true" /> {group.store.name}
          <span className="text-sm font-normal text-muted-foreground">· {group.store.location}</span>
        </Link>
        <span className="text-sm text-muted-foreground">{formatRupiah(group.subtotal)}</span>
      </header>
      <ul className="divide-y">
        {group.items.map((line) => (
          <li key={line.id}>
            <LineRow line={line} storeSlug={group.store.slug} />
          </li>
        ))}
      </ul>
     </section>
    </Card>
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
        <Link to={href} className="line-clamp-2 text-sm font-medium hover:text-primary">
          {line.productName}
        </Link>
        <p className="text-xs text-muted-foreground">Varian: {line.variantName}</p>
        <p className="text-sm font-semibold">{formatRupiah(line.unitPrice)}</p>

        {line.issue && (
          <p role="status" className="flex items-start gap-1 text-sm text-amber-800">
            <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
            {ISSUE_TEXT[line.issue](line.availableQuantity)}
            {line.issue === 'insufficient_stock' && (
              <Button
                variant="link"
                disabled={busy}
                onClick={() => run(() => update(line.id, line.availableQuantity))}
                className="ml-1 h-auto p-0 font-medium text-amber-900 underline"
              >
                Ubah ke {line.availableQuantity}
              </Button>
            )}
          </p>
        )}
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
      </div>

      <div className="flex flex-col items-end justify-between gap-2">
        <span className="text-sm font-semibold">{formatRupiah(line.lineTotal)}</span>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => run(() => remove(line.id))}
            disabled={busy}
            aria-label={`Hapus ${line.productName} dari keranjang`}
            className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
          >
            <Trash2 aria-hidden="true" />
          </Button>
          <div role="group" aria-label={`Jumlah ${line.productName}`} className="inline-flex items-center rounded-lg border">
            <Button
              variant="ghost"
              size="icon"
              aria-label="Kurangi jumlah"
              disabled={busy || !canChange || line.quantity <= 1}
              onClick={() => run(() => update(line.id, line.quantity - 1))}
            >
              <Minus aria-hidden="true" />
            </Button>
            <output aria-live="polite" className="min-w-8 text-center text-sm">
              {line.quantity}
            </output>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Tambah jumlah"
              disabled={busy || !canChange || line.quantity >= maxQuantity}
              onClick={() => run(() => update(line.id, line.quantity + 1))}
            >
              <Plus aria-hidden="true" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
