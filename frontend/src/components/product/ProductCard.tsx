import { Link } from 'react-router-dom'
import { MapPin } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { formatCompactCount, formatRupiah } from '../../lib/format'
import type { ProductSummary } from '../../types/marketplace'
import { RatingBadge } from '../shared/RatingBadge'
import { ProductImage } from './ProductImage'
import { WishlistButton } from '@/features/wishlist/WishlistButton'

export function ProductCard({ product }: { product: ProductSummary }) {
  const href = `/stores/${product.store.slug}/products/${product.slug}`
  const startsAt = product.lowestPrice !== product.highestPrice

  // The heart button is a SIBLING of the link, not inside it: a button inside
  // <a> is invalid HTML, and its clicks (also on a disabled button) would
  // open the product page. `group` sits on the wrapper so hovering the button
  // still highlights the card.
  return (
    <div className="group relative h-full">
      <Link
        to={href}
        className="block h-full rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
      >
        <Card className="h-full gap-0 py-0 transition group-hover:shadow-md group-hover:ring-primary/50">
          <div className="relative overflow-hidden">
            <ProductImage
              id={product.id}
              name={product.name}
              image={product.images[0]}
              className={`transition group-hover:scale-105 ${product.inStock ? '' : 'opacity-60'}`}
            />
            {!product.inStock && (
              <Badge className="absolute left-2 top-2 rounded-md bg-foreground/80 text-background">Stok habis</Badge>
            )}
          </div>

          <div className="flex flex-1 flex-col gap-1.5 p-3">
            <h3 className="line-clamp-2 min-h-10 text-sm text-foreground/90">{product.name}</h3>
            <p className="text-base font-semibold">
              {startsAt && <span className="mr-1 text-xs font-normal text-muted-foreground">mulai</span>}
              {formatRupiah(product.lowestPrice)}
            </p>
            <p className="flex items-center gap-1 text-xs text-muted-foreground">
              <MapPin className="size-3.5 shrink-0" aria-hidden="true" />
              <span className="truncate">
                {product.store.location} · {product.store.name}
              </span>
            </p>
            <div className="mt-auto flex items-center gap-2 pt-1">
              <RatingBadge rating={product.rating} />
              <span className="text-xs text-muted-foreground">· {formatCompactCount(product.soldCount)} terjual</span>
            </div>
          </div>
        </Card>
      </Link>
      {/* Over the image's top-right corner, on top of the link (z-10). */}
      <div className="absolute right-2 top-2 z-10">
        <WishlistButton productId={product.id} />
      </div>
    </div>
  )
}
