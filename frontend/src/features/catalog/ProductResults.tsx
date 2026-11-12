import type { ReactNode } from 'react'
import { ProductGrid } from '../../components/product/ProductGrid'
import { EmptyState, ErrorState, ProductGridSkeleton } from '../../components/shared/StateViews'
import type { AsyncState } from '../../hooks/useAsync'
import type { Paginated, ProductSummary } from '../../types/marketplace'

interface ProductResultsProps {
  state: AsyncState<Paginated<ProductSummary>>
  onRetry: () => void
  skeletonCount?: number
  emptyTitle?: string
  emptyDescription?: string
  emptyAction?: ReactNode
}

/** One place that turns a product-list request into all four UI states. */
export function ProductResults({
  state,
  onRetry,
  skeletonCount = 8,
  emptyTitle = 'Produk tidak ditemukan',
  emptyDescription = 'Coba kata kunci lain atau ubah filter.',
  emptyAction,
}: ProductResultsProps) {
  switch (state.status) {
    case 'loading':
      return <ProductGridSkeleton count={skeletonCount} />
    case 'error':
      return <ErrorState error={state.error} onRetry={onRetry} title="Gagal memuat produk" />
    case 'success':
      return state.data.items.length === 0 ? (
        <EmptyState title={emptyTitle} description={emptyDescription} action={emptyAction} />
      ) : (
        <ProductGrid products={state.data.items} />
      )
  }
}
