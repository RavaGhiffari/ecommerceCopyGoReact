import { Link } from 'react-router-dom'
import { AlertTriangle, Store } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { cn } from '@/lib/utils'
import { ProductImage } from '../../components/product/ProductImage'
import { formatRupiah } from '../../lib/format'
import type { CheckoutGroup, CheckoutLine, CheckoutLineIssue, ShippingMethod } from '../../types/checkout'
import { SHIPPING_HINT, SHIPPING_LABEL } from '../orders/status'

const ISSUE_TEXT: Record<CheckoutLineIssue, (available: number) => string> = {
  unavailable: () => 'Produk ini sudah tidak dijual.',
  out_of_stock: () => 'Stok habis.',
  insufficient_stock: (n) => `Stok tersisa ${n}.`,
  own_product: () => 'Produk dari tokomu sendiri tidak bisa dibeli.',
}

/** One store of the checkout: its lines (server prices) and a shipping choice. */
export function StoreGroupCard({
  group,
  onMethodChange,
  disabled,
}: {
  group: CheckoutGroup
  onMethodChange: (method: ShippingMethod) => void
  disabled?: boolean
}) {
  const headingId = `store-${group.store.id}`
  const shippingId = `shipping-${group.store.id}`
  return (
    <Card className="gap-0 py-0">
      <section aria-labelledby={headingId}>
        <header className="flex items-center justify-between border-b px-5 py-3">
          <Link id={headingId} to={`/stores/${group.store.slug}`} className="inline-flex items-center gap-2 font-medium hover:text-primary">
            <Store className="size-4" aria-hidden="true" /> {group.store.name}
            <span className="text-sm font-normal text-muted-foreground">· {group.store.location}</span>
          </Link>
        </header>

        <ul className="divide-y px-5">
          {group.items.map((line) => (
            <li key={line.itemId}>
              <Line line={line} storeSlug={group.store.slug} />
            </li>
          ))}
        </ul>

        <div className="border-t bg-muted/30 px-5 py-4">
          <h3 id={shippingId} className="mb-2 text-sm font-medium">
            Pengiriman dari {group.store.name}
          </h3>
          <RadioGroup
            aria-labelledby={shippingId}
            value={group.shippingMethod}
            onValueChange={(v) => onMethodChange(v as ShippingMethod)}
            disabled={disabled}
            className="gap-2 sm:grid-cols-3"
          >
            {group.shippingOptions.map((o) => {
              const id = `${group.store.id}-${o.method}`
              return (
                <label
                  key={o.method}
                  htmlFor={id}
                  className={cn(
                    'flex cursor-pointer items-start gap-2 rounded-lg border bg-background p-3 text-sm',
                    group.shippingMethod === o.method && 'border-primary bg-primary/5',
                  )}
                >
                  <RadioGroupItem id={id} value={o.method} className="mt-0.5" />
                  <span>
                    <span className="block font-medium">{SHIPPING_LABEL[o.method]}</span>
                    <span className="block text-xs text-muted-foreground">{SHIPPING_HINT[o.method]}</span>
                    <span className="block">{formatRupiah(o.fee)}</span>
                  </span>
                </label>
              )
            })}
          </RadioGroup>
        </div>

        <dl className="flex justify-between border-t px-5 py-3 text-sm">
          <dt className="text-muted-foreground">Subtotal toko (termasuk ongkir)</dt>
          <dd className="font-semibold">{formatRupiah(group.total)}</dd>
        </dl>
      </section>
    </Card>
  )
}

function Line({ line, storeSlug }: { line: CheckoutLine; storeSlug: string }) {
  return (
    <div className="flex gap-3 py-3">
      <div className="w-16 shrink-0">
        <ProductImage
          id={line.productId}
          name={line.productName}
          image={line.image ?? undefined}
          className={cn('rounded-md', line.issue && 'opacity-50')}
        />
      </div>
      <div className="min-w-0 flex-1">
        <Link to={`/stores/${storeSlug}/products/${line.productSlug}`} className="line-clamp-2 text-sm font-medium hover:text-primary">
          {line.productName}
        </Link>
        <p className="text-xs text-muted-foreground">Varian: {line.variantName}</p>
        <p className="text-sm text-muted-foreground">
          {line.quantity} × {formatRupiah(line.unitPrice)}
        </p>
        {line.issue && (
          <p role="status" className="mt-1 flex items-start gap-1 text-sm text-amber-800">
            <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
            {ISSUE_TEXT[line.issue](line.availableQuantity)}
          </p>
        )}
      </div>
      <span className={cn('text-sm font-semibold', line.issue && 'text-muted-foreground line-through')}>
        {formatRupiah(line.lineTotal)}
      </span>
    </div>
  )
}
