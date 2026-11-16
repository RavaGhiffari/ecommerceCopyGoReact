import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ChevronLeft, Store } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ItemRowSkeleton, LineSkeleton, LoadingRegion } from '../../components/shared/Skeletons'
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
    <LoadingRegion label="Memuat pesanan" className="block space-y-4">
      <LineSkeleton className="w-24" />
      <div className="flex items-center justify-between gap-3">
        <div className="space-y-2">
          <Skeleton className="h-8 w-64" />
          <LineSkeleton className="w-40" />
        </div>
        <Skeleton className="h-7 w-28 rounded-md" />
      </div>
      <div className="grid gap-4 lg:grid-cols-[1fr_20rem] lg:items-start">
        <div className="space-y-4">
          <Card className="block p-0">
            <div className="px-5 pt-5">
              <LineSkeleton className="w-40" />
            </div>
            <ItemRowSkeleton thumb="w-14" />
            <ItemRowSkeleton thumb="w-14" />
            <div className="space-y-2 border-t p-5">
              <LineSkeleton className="ml-auto w-48" />
              <LineSkeleton className="ml-auto w-40" />
            </div>
          </Card>
          <Card className="block space-y-3 p-5">
            <LineSkeleton className="w-32" />
            {[0, 1, 2].map((i) => (
              <div key={i} className="flex items-center gap-3">
                <Skeleton className="size-3 rounded-full" />
                <LineSkeleton className="w-1/2" />
              </div>
            ))}
          </Card>
        </div>
        <div className="space-y-4">
          <Card className="block space-y-3 p-5">
            <LineSkeleton className="w-1/2" />
            <Skeleton className="h-9 w-full" />
          </Card>
          <Card className="block space-y-2 p-5">
            <LineSkeleton className="w-1/3" />
            <LineSkeleton className="w-3/4" />
            <LineSkeleton className="w-full" />
            <LineSkeleton className="w-2/3" />
          </Card>
        </div>
      </div>
    </LoadingRegion>
  )
}
