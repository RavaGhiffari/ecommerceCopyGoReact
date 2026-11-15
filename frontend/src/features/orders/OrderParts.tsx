import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { MapPin, Truck } from 'lucide-react'
import { cn } from '@/lib/utils'
import { ProductImage } from '../../components/product/ProductImage'
import { formatDateTime, formatRupiah } from '../../lib/format'
import type { Order, OrderHistoryEntry, OrderItem, OrderStatus, ShippingAddress } from '../../types/orders'
import { historyNote, ORDER_STATUS_LABEL, ORDER_STATUS_TONE, SHIPPING_LABEL } from './status'

// Building blocks shared by the buyer and seller order pages.

export function OrderStatusBadge({ status, className }: { status: OrderStatus; className?: string }) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium whitespace-nowrap',
        ORDER_STATUS_TONE[status],
        className,
      )}
    >
      {ORDER_STATUS_LABEL[status]}
    </span>
  )
}

/**
 * Item rows. Names and prices are the snapshots taken at checkout.
 * `itemAction` adds something under an item (the buyer's review button).
 */
export function OrderItemList({
  items,
  storeSlug,
  itemAction,
}: {
  items: OrderItem[]
  storeSlug: string
  itemAction?: (item: OrderItem) => ReactNode
}) {
  return (
    <ul className="divide-y">
      {items.map((it) => (
        <li key={it.id} className="flex gap-3 py-3">
          <div className="w-14 shrink-0">
            <ProductImage id={it.productId} name={it.productName} image={it.image ?? undefined} className="rounded-md" />
          </div>
          <div className="min-w-0 flex-1">
            <Link
              to={`/stores/${storeSlug}/products/${it.productSlug}`}
              className="line-clamp-2 text-sm font-medium hover:text-primary"
            >
              {it.productName}
            </Link>
            <p className="text-xs text-muted-foreground">
              {it.variantName} · SKU {it.sku}
            </p>
            <p className="text-sm text-muted-foreground">
              {it.quantity} × {formatRupiah(it.unitPrice)}
            </p>
            {itemAction && <div className="mt-2">{itemAction(it)}</div>}
          </div>
          <span className="text-sm font-semibold">{formatRupiah(it.lineTotal)}</span>
        </li>
      ))}
    </ul>
  )
}

export function ShippingBlock({ order }: { order: Order }) {
  const a: ShippingAddress = order.shippingAddress
  return (
    <div className="space-y-3 text-sm">
      <div>
        <h3 className="mb-1 flex items-center gap-1.5 font-medium">
          <MapPin className="size-4 text-muted-foreground" aria-hidden="true" /> Alamat pengiriman
        </h3>
        <p>
          {a.recipientName} · <span className="text-muted-foreground">{a.phone}</span>
        </p>
        <p className="text-muted-foreground">
          {a.addressLine}, {a.city}, {a.province} {a.postalCode}
        </p>
      </div>
      <div>
        <h3 className="mb-1 flex items-center gap-1.5 font-medium">
          <Truck className="size-4 text-muted-foreground" aria-hidden="true" /> Pengiriman
        </h3>
        <p>
          {SHIPPING_LABEL[order.shippingMethod]} · {formatRupiah(order.shippingFee)}
        </p>
      </div>
    </div>
  )
}

export function OrderTotals({ order }: { order: Order }) {
  return (
    <dl className="space-y-1.5 text-sm">
      <div className="flex justify-between">
        <dt className="text-muted-foreground">Subtotal barang</dt>
        <dd>{formatRupiah(order.itemsSubtotal)}</dd>
      </div>
      <div className="flex justify-between">
        <dt className="text-muted-foreground">Ongkos kirim</dt>
        <dd>{formatRupiah(order.shippingFee)}</dd>
      </div>
      <div className="flex justify-between border-t pt-2 text-base font-semibold">
        <dt>Total</dt>
        <dd>{formatRupiah(order.total)}</dd>
      </div>
    </dl>
  )
}

/** Status history, newest at the top. */
export function OrderTimeline({ history }: { history: OrderHistoryEntry[] }) {
  const entries = [...history].reverse()
  return (
    <ol aria-label="Riwayat status" className="space-y-0">
      {entries.map((h, i) => (
        <li key={`${h.toStatus}-${h.createdAt}`} className="relative flex gap-3 pb-4 last:pb-0">
          {i < entries.length - 1 && (
            <span aria-hidden="true" className="absolute top-4 left-[5px] h-full w-px bg-border" />
          )}
          <span
            aria-hidden="true"
            className={cn('relative mt-1.5 size-[11px] shrink-0 rounded-full', i === 0 ? 'bg-primary' : 'bg-muted-foreground/40')}
          />
          <div className="min-w-0">
            <p className={cn('text-sm', i === 0 && 'font-semibold')}>{ORDER_STATUS_LABEL[h.toStatus]}</p>
            {h.note && <p className="text-sm text-muted-foreground">{historyNote(h.note)}</p>}
            <p className="text-xs text-muted-foreground">
              <time dateTime={h.createdAt}>{formatDateTime(h.createdAt)}</time>
            </p>
          </div>
        </li>
      ))}
    </ol>
  )
}
