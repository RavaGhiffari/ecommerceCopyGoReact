import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ChevronLeft, Store } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { formatDateTime } from '../../lib/format'
import type { Order, OrderItem } from '../../types/orders'
import { OrderItemList, OrderStatusBadge, OrderTimeline, OrderTotals, ShippingBlock } from './OrderParts'

/** Shared by /orders/:id (buyer) and /seller/orders/:id (seller). `actions` differ. */
export function OrderDetailLayout({
  order,
  back,
  actions,
  itemAction,
}: {
  order: Order
  back: { to: string; label: string }
  actions: ReactNode
  itemAction?: (item: OrderItem) => ReactNode
}) {
  return (
    <div className="space-y-4">
      <Link to={back.to} className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-primary">
        <ChevronLeft className="size-4" aria-hidden="true" /> {back.label}
      </Link>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">{order.orderNumber}</h1>
          <p className="text-sm text-muted-foreground">Dibuat {formatDateTime(order.createdAt)}</p>
        </div>
        <OrderStatusBadge status={order.status} className="px-3 py-1 text-sm" />
      </div>

      <div className="grid gap-4 lg:grid-cols-[1fr_20rem] lg:items-start">
        <div className="space-y-4">
          <Card className="block p-5">
            <h2 className="flex items-center gap-2 font-semibold">
              <Store className="size-4" aria-hidden="true" />
              <Link to={`/stores/${order.store.slug}`} className="hover:text-primary">
                {order.store.name}
              </Link>
            </h2>
            <OrderItemList items={order.items} storeSlug={order.store.slug} itemAction={itemAction} />
            <div className="border-t pt-3">
              <OrderTotals order={order} />
            </div>
          </Card>

          <Card className="block p-5">
            <h2 className="mb-3 font-semibold">Riwayat status</h2>
            <OrderTimeline history={order.history ?? []} />
          </Card>
        </div>

        <div className="space-y-4">
          <Card className="block p-5">{actions}</Card>
          <Card className="block p-5">
            <ShippingBlock order={order} />
          </Card>
        </div>
      </div>
    </div>
  )
}

export function OrderDetailSkeleton() {
  return (
    <div className="space-y-4" role="status" aria-label="Memuat pesanan">
      <Skeleton className="h-10 w-64" />
      <div className="grid gap-4 lg:grid-cols-[1fr_20rem]">
        <Skeleton className="h-72 rounded-xl" />
        <Skeleton className="h-48 rounded-xl" />
      </div>
    </div>
  )
}
