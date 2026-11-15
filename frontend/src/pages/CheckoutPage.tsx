import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AlertTriangle, Info, Loader2, RotateCw } from 'lucide-react'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { FormAlert } from '../components/shared/TextField'
import { messageFor } from '../features/auth/serverErrors'
import { useCart } from '../features/cart/cartContext'
import { AddressPicker } from '../features/checkout/AddressPicker'
import { StoreGroupCard } from '../features/checkout/StoreGroupCard'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { ApiError } from '../lib/api'
import { formatRupiah } from '../lib/format'
import { listAddresses } from '../services/addresses'
import { placeCheckout, quoteCheckout } from '../services/checkout'
import type { Address } from '../types/address'
import type { Cart } from '../types/cart'
import type { CheckoutInput, Quote, ShippingMethod } from '../types/checkout'

/**
 * /checkout (behind RequireAuth). The whole cart is checked out (decision Q3).
 * Every amount on this page comes from POST /checkout/quote; the client never
 * adds up money. "Buat pesanan" sends exactly what the last quote showed.
 */
export function CheckoutPage() {
  usePageTitle('Checkout')
  const cart = useCart()
  const addresses = useAsync(() => listAddresses(), 'checkout:addresses')
  // Keep the previous list while it reloads (after adding an address), so
  // the form below stays mounted and keeps its choices.
  const list = addresses.state.status === 'success' ? addresses.state.data : addresses.lastData

  if (cart.status === 'error' && cart.error) {
    return <ErrorState error={cart.error} onRetry={cart.reload} title="Gagal memuat keranjang" />
  }
  if (addresses.state.status === 'error' && !list) {
    return <ErrorState error={addresses.state.error} onRetry={addresses.reload} title="Gagal memuat alamat" />
  }
  if (cart.status !== 'ready' || !list) {
    return (
      <div className="grid gap-6 lg:grid-cols-[1fr_20rem]" role="status" aria-label="Memuat checkout">
        <div className="space-y-4">
          <Skeleton className="h-40 rounded-xl" />
          <Skeleton className="h-56 rounded-xl" />
        </div>
        <Skeleton className="h-64 rounded-xl" />
      </div>
    )
  }
  if (!cart.cart || cart.cart.stores.length === 0) {
    return (
      <EmptyState
        title="Tidak ada yang bisa di-checkout"
        description="Keranjangmu kosong. Tambahkan produk dulu."
        action={
          <Button size="lg" asChild>
            <Link to="/products">Mulai belanja</Link>
          </Button>
        }
      />
    )
  }
  return <CheckoutForm cart={cart.cart} addresses={list} reloadAddresses={addresses.reload} />
}

function CheckoutForm({
  cart,
  addresses,
  reloadAddresses,
}: {
  cart: Cart
  addresses: Address[]
  reloadAddresses: () => void
}) {
  const navigate = useNavigate()
  const { reload: reloadCart } = useCart()

  // Generated once per visit. A double click or a retry after a timeout sends
  // the same key, and the server answers with the checkout it already made.
  const [idempotencyKey] = useState(() => crypto.randomUUID())

  const [chosenAddress, setChosenAddress] = useState<string | null>(null)
  const fallback = addresses.find((a) => a.isDefault) ?? addresses[0]
  const addressId = chosenAddress ?? fallback?.id ?? null

  // Shipping choices per store. Unchosen stores are quoted as "regular".
  const [choices, setChoices] = useState<Record<string, ShippingMethod>>({})
  const storeIds = new Set(cart.stores.map((g) => g.store.id))
  const shipping = Object.fromEntries(Object.entries(choices).filter(([id]) => storeIds.has(id)))
  const input: CheckoutInput | null = addressId ? { addressId, shipping } : null

  // Re-quote whenever the address, a shipping choice or the cart changes.
  const cartSignature = cart.stores.flatMap((g) => g.items.map((i) => `${i.id}:${i.quantity}`)).join(',')
  const quote = useAsync<Quote | null>(
    () => (input ? quoteCheckout(input) : Promise.resolve(null)),
    `quote:${JSON.stringify(input)}:${cartSignature}`,
  )
  const shown = quote.state.status === 'success' ? quote.state.data : (quote.lastData ?? null)
  const recalculating = quote.state.status === 'loading'

  const [placing, setPlacing] = useState(false)
  const [placeError, setPlaceError] = useState<string | null>(null)

  const chooseAddress = (id: string) => {
    setChosenAddress(id)
    // Same Day depends on the destination city: start from "regular" again.
    setChoices({})
    setPlaceError(null)
  }

  const place = async () => {
    if (!shown) return
    setPlacing(true)
    setPlaceError(null)
    try {
      const placed = await placeCheckout(
        {
          addressId: shown.address.id,
          shipping: Object.fromEntries(shown.stores.map((g) => [g.store.id, g.shippingMethod])),
        },
        idempotencyKey,
      )
      toast.success(
        placed.orders.length > 1
          ? `${placed.orders.length} pesanan dibuat. Selesaikan pembayaran.`
          : 'Pesanan dibuat. Selesaikan pembayaran.',
      )
      // Leave first, then refresh the (now emptier) cart, so this page never
      // renders the empty-cart state in between.
      await navigate(`/checkout/${placed.id}`, { replace: true, flushSync: true })
      reloadCart()
    } catch (err) {
      setPlaceError(messageFor(err))
      if (err instanceof ApiError) {
        if (err.code === 'CART_HAS_ISSUES') quote.reload() // the quote marks the lines
        if (err.code === 'CART_EMPTY') reloadCart()
        if (err.code === 'SHIPPING_METHOD_NOT_ALLOWED') setChoices({})
        if (err.code === 'ADDRESS_NOT_FOUND') {
          setChosenAddress(null)
          reloadAddresses()
        }
      }
    } finally {
      setPlacing(false)
    }
  }

  const quoteError = quote.state.status === 'error' ? quote.state.error : null
  const canPlace = !!shown && !recalculating && !quoteError && !shown.hasIssues && !placing

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_20rem] lg:items-start">
      <div className="space-y-4">
        <h1 className="text-2xl font-semibold">Checkout</h1>

        <AddressPicker
          addresses={addresses}
          value={addressId}
          onChange={chooseAddress}
          onAdded={(saved) => {
            chooseAddress(saved.id)
            reloadAddresses()
          }}
          disabled={placing}
        />

        {shown?.hasIssues && (
          <Alert role="alert" className="border-amber-200 bg-amber-50 text-amber-900">
            <AlertTriangle aria-hidden="true" />
            <AlertDescription className="text-amber-900">
              Beberapa barang tidak bisa dibeli saat ini.{' '}
              <Link to="/cart" className="font-medium underline">
                Perbaiki di keranjang
              </Link>{' '}
              sebelum membuat pesanan.
            </AlertDescription>
          </Alert>
        )}

        {quoteError && (
          <QuoteErrorAlert
            error={quoteError}
            onReset={() => {
              setChoices({})
              setChosenAddress(null)
              quote.reload()
            }}
          />
        )}

        {!addressId && (
          <p className="text-sm text-muted-foreground">Pilih atau tambahkan alamat untuk melihat ongkos kirim.</p>
        )}

        {shown?.stores.map((group) => (
          <StoreGroupCard
            key={group.store.id}
            group={group}
            disabled={recalculating || placing}
            onMethodChange={(method) => {
              setChoices((prev) => ({ ...prev, [group.store.id]: method }))
              setPlaceError(null)
            }}
          />
        ))}
      </div>

      <aside className="lg:sticky lg:top-32" aria-label="Ringkasan pesanan">
        <Card className="gap-3 p-5">
          <h2 className="flex items-center justify-between text-lg font-semibold">
            Ringkasan pesanan
            {recalculating && shown && (
              <Loader2 className="size-4 animate-spin text-muted-foreground" aria-label="Menghitung ulang" />
            )}
          </h2>
          {shown ? (
            <dl className="space-y-1.5 text-sm" aria-busy={recalculating}>
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Total harga barang</dt>
                <dd>{formatRupiah(shown.itemsTotal)}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Total ongkos kirim</dt>
                <dd>{formatRupiah(shown.shippingTotal)}</dd>
              </div>
              <div className="flex justify-between border-t pt-2 text-base font-semibold">
                <dt>Total tagihan</dt>
                <dd>{formatRupiah(shown.grandTotal)}</dd>
              </div>
            </dl>
          ) : (
            <Skeleton className="h-20" aria-hidden="true" />
          )}

          <FormAlert message={placeError} />

          <Button size="lg" className="h-10 w-full font-semibold" disabled={!canPlace} onClick={() => void place()}>
            {placing ? 'Membuat pesanan…' : 'Buat pesanan'}
          </Button>
          <p className="flex gap-2 text-xs text-muted-foreground">
            <Info className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
            Stok dipesan untukmu sampai batas waktu pembayaran. Harga dan stok dicek ulang saat pesanan dibuat.
          </p>
        </Card>
      </aside>
    </div>
  )
}

function QuoteErrorAlert({ error, onReset }: { error: ApiError; onReset: () => void }) {
  return (
    <Alert role="alert" variant="destructive" className="border-destructive/30 bg-destructive/5">
      <AlertTriangle aria-hidden="true" />
      <AlertDescription>
        <p>{messageFor(error)}</p>
        <Button variant="outline" size="sm" className="mt-2" onClick={onReset}>
          <RotateCw aria-hidden="true" /> Hitung ulang
        </Button>
      </AlertDescription>
    </Alert>
  )
}
