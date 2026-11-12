import { Link } from 'react-router-dom'
import { MapPin } from 'lucide-react'
import { formatCompactCount, formatRupiah } from '../../lib/format'
import type { ProductSummary } from '../../types/marketplace'
import { RatingBadge } from '../shared/RatingBadge'
import { ProductImage } from './ProductImage'

export function ProductCard({ product }: { product: ProductSummary }) {
  const href = `/stores/${product.store.slug}/products/${product.slug}`
  const startsAt = product.lowestPrice !== product.highestPrice

  return (
    <Link
      to={href}
      className="group flex h-full flex-col overflow-hidden rounded-xl border border-slate-200 bg-white transition hover:border-emerald-400 hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
    >
      <div className="relative overflow-hidden">
        <ProductImage
          id={product.id}
          name={product.name}
          image={product.images[0]}
          className={`transition group-hover:scale-105 ${product.inStock ? '' : 'opacity-60'}`}
        />
        {!product.inStock && (
          <span className="absolute left-2 top-2 rounded bg-slate-900/80 px-2 py-0.5 text-xs font-medium text-white">
            Stok habis
          </span>
        )}
      </div>

      <div className="flex flex-1 flex-col gap-1.5 p-3">
        <h3 className="line-clamp-2 min-h-10 text-sm text-slate-800">{product.name}</h3>
        <p className="text-base font-semibold text-slate-900">
          {startsAt && <span className="mr-1 text-xs font-normal text-slate-500">mulai</span>}
          {formatRupiah(product.lowestPrice)}
        </p>
        <p className="flex items-center gap-1 text-xs text-slate-500">
          <MapPin className="size-3.5 shrink-0" aria-hidden="true" />
          <span className="truncate">
            {product.store.location} · {product.store.name}
          </span>
        </p>
        <div className="mt-auto flex items-center gap-2 pt-1">
          <RatingBadge rating={product.rating} />
          <span className="text-xs text-slate-500">· {formatCompactCount(product.soldCount)} terjual</span>
        </div>
      </div>
    </Link>
  )
}
