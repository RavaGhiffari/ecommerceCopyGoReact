import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { CheckCircle2, Clock, CreditCard, RotateCw, Store, XCircle } from 'lucide-react'
import { toast } from 'sonner'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ErrorState } from '../components/shared/StateViews'
import { FormAlert } from '../components/shared/TextField'
import { messageFor } from '../features/auth/serverErrors'
import { OrderItemList, OrderStatusBadge } from '../features/orders/OrderParts'
import { CHECKOUT_STATUS_LABEL, PAYMENT_STATUS_LABEL, SHIPPING_LABEL } from '../features/orders/status'
import { useAsyncResource } from '../hooks/useAsync'
import { useNow } from '../hooks/useNow'
import { usePageTitle } from '../hooks/usePageTitle'
import { ApiError } from '../lib/api'
import { formatCountdown, formatDateTime, formatRupiah } from '../lib/format'
import { cancelCheckout, getCheckout, simulatePayment } from '../services/checkout'
import type { Checkout, PaymentOutcome } from '../types/checkout'

/** /checkout/:id (behind RequireAuth): pay (simulated) or cancel a checkout. */
export function PaymentPage() {
  const { id = '' } = useParams()
  // key: a different checkout id starts from a clean state.
  return <Payment key={id} id={id} />
}

function Payment({ id }: { id: string }) {
  usePageTitle('Pembayaran')
  const { state, replace, reload } = useAsyncResource(() => getCheckout(id), `checkout:${id}`)

  if (state.status === 'loading') {
    return (
      <div className="mx-auto max-w-3xl space-y-4" role="status" aria-label="Memuat pembayaran">
        <Skeleton className="h-48 rounded-xl" />
        <Skeleton className="h-40 rounded-xl" />
      </div>
    )
  }
  if (state.status === 'error') {
    return <ErrorState error={state.error} onRetry={reload} title="Gagal memuat pembayaran" />
  }
  return <PaymentView checkout={state.data} onChange={replace} onReload={reload} />
}

function PaymentView({
  checkout: co,
  onChange,
  onReload,
}: {
  checkout: Checkout
  onChange: (c: Checkout) => void
  onReload: () => void
}) {
  const pending = co.status === 'pending_payment'
  const now = useNow(pending)
  const remaining = new Date(co.expiresAt).getTime() - now
  const timeUp = pending && remaining <= 0

  // One gateway notification id per outcome for this page visit: clicking
  // again resends the SAME event, which the server treats as a repeat.
  const [eventIds] = useState(() => ({ success: crypto.randomUUID(), failure: crypto.randomUUID() }))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = async (action: () => Promise<Checkout>, done: (c: Checkout) => void) => {
    setBusy(true)
    setError(null)
    try {
      const updated = await action()
      onChange(updated)
      done(updated)
    } catch (err) {
      // The checkout changed on the server (expired, paid in another tab):
      // say so in a toast, because reloading replaces this view.
      if (err instanceof ApiError && (err.code === 'CHECKOUT_EXPIRED' || err.code === 'CHECKOUT_NOT_PENDING')) {
        toast.error(messageFor(err))
        onReload()
      } else {
        setError(messageFor(err))
      }
    } finally {
      setBusy(false)
    }
  }

  const pay = (outcome: PaymentOutcome) =>
    run(
      () => simulatePayment(co.payment.id, outcome, eventIds[outcome]),
      (c) =>
        c.status === 'paid'
          ? toast.success('Pembayaran berhasil. Penjual akan memproses pesananmu.')
          : toast.error('Pembayaran gagal. Pesanan dibatalkan dan stok dikembalikan.'),
    )

  const cancel = () => run(() => cancelCheckout(co.id), () => toast.success('Checkout dibatalkan.'))

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <h1 className="text-2xl font-semibold">Pembayaran</h1>

      <Card className="block p-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-sm text-muted-foreground">Total tagihan</p>
            <p className="text-3xl font-bold" data-testid="grand-total">
              {formatRupiah(co.grandTotal)}
            </p>
            <p className="mt-1 text-sm text-muted-foreground">
              Barang {formatRupiah(co.itemsTotal)} · Ongkir {formatRupiah(co.shippingTotal)}
            </p>
          </div>
          <StatusPill checkout={co} />
        </div>

        <dl className="mt-4 grid gap-2 border-t pt-4 text-sm sm:grid-cols-2">
          <div>
            <dt className="text-muted-foreground">Referensi pembayaran</dt>
            <dd className="font-mono">{co.payment.reference}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Status pembayaran</dt>
            <dd>{PAYMENT_STATUS_LABEL[co.payment.status]}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Dibuat</dt>
            <dd>{formatDateTime(co.createdAt)}</dd>
          </div>
          {co.paidAt && (
            <div>
              <dt className="text-muted-foreground">Waktu bayar</dt>
              <dd>{formatDateTime(co.paidAt)}</dd>
            </div>
          )}
        </dl>

        {pending && !timeUp && (
          <div className="mt-4 rounded-lg bg-amber-50 p-4 text-amber-950">
            <p className="flex items-center gap-2 text-sm">
              <Clock className="size-4" aria-hidden="true" /> Bayar sebelum {formatDateTime(co.expiresAt)}
            </p>
            <p className="mt-1 text-2xl font-semibold tabular-nums" aria-label="Sisa waktu pembayaran" role="timer">
              {formatCountdown(remaining)}
            </p>
          </div>
        )}

        {timeUp && (
          <Alert role="alert" className="mt-4 border-amber-200 bg-amber-50 text-amber-900">
            <Clock aria-hidden="true" />
            <AlertDescription className="text-amber-900">
              <p>Waktu pembayaran sudah habis. Pesanan akan dibatalkan otomatis dan stok dikembalikan.</p>
              <Button variant="outline" size="sm" className="mt-2" onClick={onReload}>
                <RotateCw aria-hidden="true" /> Muat ulang status
              </Button>
            </AlertDescription>
          </Alert>
        )}

        <FormAlert message={error} />

        {pending && (
          <div className="mt-4 space-y-3 border-t pt-4">
            <p className="flex items-center gap-2 text-sm text-muted-foreground">
              <CreditCard className="size-4" aria-hidden="true" />
              Pembayaran disimulasikan, tidak ada uang sungguhan yang berpindah.
            </p>
            <div className="flex flex-wrap gap-2">
              <Button size="lg" disabled={busy || timeUp} onClick={() => void pay('success')}>
                <CheckCircle2 aria-hidden="true" /> Simulasikan bayar berhasil
              </Button>
              <Button variant="outline" size="lg" disabled={busy || timeUp} onClick={() => void pay('failure')}>
                <XCircle aria-hidden="true" /> Simulasikan bayar gagal
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button variant="ghost" size="lg" className="text-destructive hover:text-destructive" disabled={busy}>
                    Batalkan checkout
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Batalkan checkout ini?</AlertDialogTitle>
                    <AlertDialogDescription>
                      Semua pesanan di checkout ini ({co.orders.length} toko) dibatalkan dan stoknya dikembalikan.
                      Barangnya tidak kembali ke keranjang.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Jangan batalkan</AlertDialogCancel>
                    <AlertDialogAction variant="destructive" onClick={() => void cancel()}>
                      Ya, batalkan
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          </div>
        )}

        {co.status === 'paid' && (
          <p className="mt-4 text-sm">
            Pembayaran diterima.{' '}
            <Link to="/orders" className="font-medium text-primary underline">
              Lihat pesanan saya
            </Link>
          </p>
        )}
      </Card>

      <section aria-labelledby="orders-heading" className="space-y-3">
        <h2 id="orders-heading" className="text-lg font-semibold">
          Pesanan per toko ({co.orders.length})
        </h2>
        {co.orders.map((o) => (
          <Card key={o.id} className="block p-5">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="flex items-center gap-2 font-medium">
                <Store className="size-4" aria-hidden="true" /> {o.store.name}
              </p>
              <OrderStatusBadge status={o.status} />
            </div>
            <p className="text-sm text-muted-foreground">
              {o.orderNumber} · {SHIPPING_LABEL[o.shippingMethod]} {formatRupiah(o.shippingFee)}
            </p>
            <OrderItemList items={o.items} storeSlug={o.store.slug} />
            <div className="flex items-center justify-between border-t pt-3">
              <Link to={`/orders/${o.id}`} className="text-sm font-medium text-primary hover:underline">
                Lihat detail pesanan
              </Link>
              <span className="font-semibold">{formatRupiah(o.total)}</span>
            </div>
          </Card>
        ))}
      </section>
    </div>
  )
}

function StatusPill({ checkout: co }: { checkout: Checkout }) {
  const tone =
    co.status === 'paid'
      ? 'bg-emerald-100 text-emerald-900'
      : co.status === 'pending_payment'
        ? 'bg-amber-100 text-amber-900'
        : 'bg-muted text-muted-foreground'
  return (
    <span className={`rounded-full px-3 py-1 text-sm font-medium ${tone}`} data-testid="checkout-status">
      {CHECKOUT_STATUS_LABEL[co.status]}
    </span>
  )
}
