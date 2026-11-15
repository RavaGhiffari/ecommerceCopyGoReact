import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
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
import { Button } from '@/components/ui/button'
import { ErrorState } from '../components/shared/StateViews'
import { FormAlert } from '../components/shared/TextField'
import { messageFor } from '../features/auth/serverErrors'
import { OrderDetailLayout, OrderDetailSkeleton } from '../features/orders/OrderDetailLayout'
import { buyerCanRefund } from '../features/orders/status'
import { useAsyncResource } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { ApiError } from '../lib/api'
import { formatRupiah } from '../lib/format'
import { Star } from 'lucide-react'
import { Stars } from '../components/shared/Stars'
import { ReviewDialog } from '../features/reviews/ReviewDialog'
import { cancelMyOrder, getMyOrder } from '../services/orders'
import type { Order, OrderItem } from '../types/orders'

/** /orders/:id (behind RequireAuth). Another buyer's order is a 404 from the API. */
export function OrderDetailPage() {
  const { id = '' } = useParams()
  return <BuyerOrder key={id} id={id} />
}

function BuyerOrder({ id }: { id: string }) {
  usePageTitle('Detail pesanan')
  const { state, replace, reload } = useAsyncResource(() => getMyOrder(id), `my-order:${id}`)
  const [reviewing, setReviewing] = useState<OrderItem | null>(null)

  if (state.status === 'loading') return <OrderDetailSkeleton />
  if (state.status === 'error') return <ErrorState error={state.error} onRetry={reload} title="Gagal memuat pesanan" />

  const order = state.data
  return (
    <>
      <OrderDetailLayout
        order={order}
        back={{ to: '/orders', label: 'Pesanan saya' }}
        actions={<BuyerActions order={order} onChange={replace} onReload={reload} />}
        // Reviews are allowed only once the order is delivered (the API decides).
        itemAction={order.status === 'delivered' ? (item) => <ReviewButton item={item} onOpen={setReviewing} /> : undefined}
      />
      {reviewing && (
        <ReviewDialog
          open
          onOpenChange={(open) => !open && setReviewing(null)}
          target={{ orderItemId: reviewing.id, productName: reviewing.productName, variantName: reviewing.variantName }}
          existing={reviewing.review ?? undefined}
          onSaved={() => {
            toast.success(reviewing.review ? 'Ulasan diperbarui.' : 'Terima kasih! Ulasanmu sudah tersimpan.')
            setReviewing(null)
            // Refetch quietly (no skeleton): the items now carry the review.
            getMyOrder(order.id).then(replace, reload)
          }}
        />
      )}
    </>
  )
}

function ReviewButton({ item, onOpen }: { item: OrderItem; onOpen: (item: OrderItem) => void }) {
  if (!item.review) {
    return (
      <Button size="sm" onClick={() => onOpen(item)}>
        <Star aria-hidden="true" /> Beri ulasan
      </Button>
    )
  }
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Stars value={item.review.rating} />
      <Button size="sm" variant="outline" onClick={() => onOpen(item)}>
        Ubah ulasan
      </Button>
    </div>
  )
}

function BuyerActions({
  order,
  onChange,
  onReload,
}: {
  order: Order
  onChange: (o: Order) => void
  onReload: () => void
}) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const refund = async () => {
    setBusy(true)
    setError(null)
    try {
      onChange(await cancelMyOrder(order.id))
      toast.success('Pesanan dibatalkan. Dana dikembalikan (simulasi).')
    } catch (err) {
      // e.g. the seller started processing a moment ago: show the new status.
      if (err instanceof ApiError && err.code === 'INVALID_STATUS_TRANSITION') {
        toast.error('Pesanan sudah diproses penjual, jadi tidak bisa dibatalkan lagi.')
        onReload()
      } else {
        setError(messageFor(err))
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="space-y-3">
      <h2 className="font-semibold">Tindakan</h2>
      <FormAlert message={error} />

      {order.status === 'pending_payment' && (
        <>
          <p className="text-sm text-muted-foreground">Pesanan ini menunggu pembayaran.</p>
          <Button className="w-full" asChild>
            <Link to={`/checkout/${order.checkoutId}`}>Lanjut ke pembayaran</Link>
          </Button>
        </>
      )}

      {buyerCanRefund(order.status) && (
        <>
          <p className="text-sm text-muted-foreground">
            Kamu masih bisa membatalkan pesanan ini sebelum penjual memprosesnya.
          </p>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="outline" className="w-full text-destructive hover:text-destructive" disabled={busy}>
                Batalkan pesanan
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Batalkan pesanan {order.orderNumber}?</AlertDialogTitle>
                <AlertDialogDescription>
                  Dana {formatRupiah(order.total)} dikembalikan (simulasi) dan stok dikembalikan ke toko. Pesanan dari
                  toko lain di checkout yang sama tidak terpengaruh.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Jangan batalkan</AlertDialogCancel>
                <AlertDialogAction variant="destructive" onClick={() => void refund()}>
                  Ya, batalkan
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </>
      )}

      {!buyerCanRefund(order.status) && order.status !== 'pending_payment' && (
        <p className="text-sm text-muted-foreground">{BUYER_NOTE[order.status] ?? 'Tidak ada tindakan.'}</p>
      )}
    </div>
  )
}

const BUYER_NOTE: Partial<Record<Order['status'], string>> = {
  processing: 'Penjual sedang menyiapkan pesananmu.',
  shipped: 'Pesanan sedang dalam perjalanan.',
  delivered: 'Pesanan selesai. Terima kasih sudah berbelanja! Beri ulasan untuk setiap barang di daftar barang.',
  cancelled: 'Pesanan ini dibatalkan.',
  refunded: 'Pesanan dibatalkan dan dananya dikembalikan.',
}
