import { Link, useSearchParams } from 'react-router-dom'
import { Store } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { LineSkeleton, LoadingRegion } from '../components/shared/Skeletons'
import { ProductImage } from '../components/product/ProductImage'
import { Pagination } from '../components/shared/Pagination'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { OrderStatusBadge } from '../features/orders/OrderParts'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatDateTime, formatRupiah } from '../lib/format'
import { listMyOrders } from '../services/orders'
import type { Order } from '../types/orders'

/** /orders (behind RequireAuth): the buyer's orders, newest first. */
export function OrdersPage() {
  usePageTitle('Pesanan saya')
  const [params, setParams] = useSearchParams()
  const page = Math.max(1, Number(params.get('page')) || 1)
  const { state, reload } = useAsync(() => listMyOrders(page), `my-orders:${page}`)

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Pesanan saya</h1>

      {state.status === 'loading' && (
        <LoadingRegion label="Memuat pesanan" className="space-y-3">
          {[0, 1, 2].map((i) => (
            <OrderCardSkeleton key={i} />
          ))}
        </LoadingRegion>
      )}
      {state.status === 'error' && <ErrorState error={state.error} onRetry={reload} title="Gagal memuat pesanan" />}
      {state.status === 'success' && state.data.items.length === 0 && (
        <EmptyState
          title="Belum ada pesanan"
          description="Pesanan yang kamu buat akan muncul di sini."
          action={
            <Button size="lg" asChild>
              <Link to="/products">Mulai belanja</Link>
            </Button>
          }
        />
      )}
      {state.status === 'success' && state.data.items.length > 0 && (
        <>
          <ul className="space-y-3">
            {state.data.items.map((o) => (
              <li key={o.id}>
                <OrderCard order={o} />
              </li>
            ))}
          </ul>
          <Pagination
            page={state.data.page}
            totalPages={state.data.totalPages}
            onPageChange={(p) => setParams(p > 1 ? { page: String(p) } : {})}
          />
        </>
      )}
    </div>
  )
}

/** Same three bands as OrderCard: store + status, first item, number + buttons. */
function OrderCardSkeleton() {
  return (
    <Card className="block p-5">
      <div className="flex items-center justify-between gap-2 border-b pb-3">
        <LineSkeleton className="w-56" />
        <Skeleton className="h-5 w-24 rounded-md" />
      </div>
      <div className="flex gap-3 py-3">
        <Skeleton className="aspect-square w-14 shrink-0 rounded-md" />
        <div className="flex-1 space-y-2">
          <LineSkeleton className="w-2/3" />
          <LineSkeleton className="h-3 w-1/4" />
        </div>
      </div>
      <div className="flex items-center justify-between gap-2 border-t pt-3">
        <LineSkeleton className="w-64" />
        <Skeleton className="h-8 w-24" />
      </div>
    </Card>
  )
}

function OrderCard({ order: o }: { order: Order }) {
  const first = o.items[0]
  const more = o.items.length - 1
  return (
    <Card className="block p-5">
      <article aria-label={`Pesanan ${o.orderNumber}`}>
        <div className="flex flex-wrap items-center justify-between gap-2 border-b pb-3">
          <p className="flex items-center gap-2 text-sm">
            <Store className="size-4" aria-hidden="true" />
            <span className="font-medium">{o.store.name}</span>
            <span className="text-muted-foreground">· {formatDateTime(o.createdAt)}</span>
          </p>
          <OrderStatusBadge status={o.status} />
        </div>

        {first && (
          <div className="flex gap-3 py-3">
            <div className="w-14 shrink-0">
              <ProductImage id={first.productId} name={first.productName} image={first.image ?? undefined} className="rounded-md" />
            </div>
            <div className="min-w-0 flex-1 text-sm">
              <p className="line-clamp-1 font-medium">{first.productName}</p>
              <p className="text-muted-foreground">
                {first.quantity} × {formatRupiah(first.unitPrice)}
              </p>
              {more > 0 && <p className="text-muted-foreground">+{more} barang lainnya</p>}
            </div>
          </div>
        )}

        <div className="flex flex-wrap items-center justify-between gap-2 border-t pt-3">
          <p className="text-sm">
            <span className="text-muted-foreground">{o.orderNumber} · Total </span>
            <span className="font-semibold">{formatRupiah(o.total)}</span>
          </p>
          <div className="flex gap-2">
            {o.status === 'pending_payment' && (
              <Button size="sm" asChild>
                <Link to={`/checkout/${o.checkoutId}`}>Bayar</Link>
              </Button>
            )}
            <Button variant="outline" size="sm" asChild>
              <Link to={`/orders/${o.id}`}>Lihat detail</Link>
            </Button>
          </div>
        </div>
      </article>
    </Card>
  )
}
