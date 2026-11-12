import { Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import type { ProductQuery } from '../../types/marketplace'
import { useProducts } from './hooks'
import { ProductResults } from './ProductResults'

/** A titled row of products for the homepage (popular, newest, ...). */
export function ProductSection({
  title,
  query,
  seeAllTo,
}: {
  title: string
  query: ProductQuery
  seeAllTo: string
}) {
  const { state, reload } = useProducts(query)

  return (
    <section aria-labelledby={`section-${query.sort}`}>
      <div className="mb-3 flex items-end justify-between">
        <h2 id={`section-${query.sort}`} className="text-xl font-semibold text-slate-900">
          {title}
        </h2>
        <Link
          to={seeAllTo}
          className="inline-flex items-center text-sm font-medium text-emerald-700 hover:text-emerald-800"
        >
          Lihat semua <ChevronRight className="size-4" aria-hidden="true" />
        </Link>
      </div>
      <ProductResults state={state} onRetry={reload} skeletonCount={query.pageSize} />
    </section>
  )
}
