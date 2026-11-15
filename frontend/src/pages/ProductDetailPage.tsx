import { Skeleton } from '@/components/ui/skeleton'
import { Link, useParams } from 'react-router-dom'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { useProduct, useRelatedProducts } from '../features/catalog/hooks'
import { ProductDetailView } from '../features/catalog/ProductDetailView'
import { ProductResults } from '../features/catalog/ProductResults'
import { ProductReviewsSection } from '../features/reviews/ProductReviewsSection'
import type { AsyncState } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import type { Paginated, ProductSummary } from '../types/marketplace'

export function ProductDetailPage() {
  const { storeSlug = '', productSlug = '' } = useParams()
  const product = useProduct(storeSlug, productSlug)

  usePageTitle(product.state.status === 'success' ? product.state.data.name : 'Produk')

  switch (product.state.status) {
    case 'loading':
      return (
        <div className="grid gap-8 md:grid-cols-2" role="status" aria-label="Memuat produk">
          <Skeleton className="aspect-square rounded-xl" />
          <div className="space-y-4">
            <Skeleton className="h-8 w-3/4" />
            <Skeleton className="h-5 w-1/3" />
            <Skeleton className="h-10 w-1/2" />
            <Skeleton className="h-24" />
          </div>
        </div>
      )
    case 'error':
      return product.state.error.status === 404 ? (
        <EmptyState
          title="Produk tidak ditemukan"
          description="Produk ini tidak ada, sudah diarsipkan, atau alamatnya salah."
          action={
            <Link to="/products" className="text-sm font-medium text-primary hover:underline">
              Lihat semua produk
            </Link>
          }
        />
      ) : (
        <ErrorState error={product.state.error} onRetry={product.reload} title="Gagal memuat produk" />
      )
    case 'success':
      return (
        <div className="space-y-12">
          <ProductDetailView key={product.state.data.id} product={product.state.data} />
          <ProductReviewsSection key={`reviews-${product.state.data.id}`} productId={product.state.data.id} />
          <RelatedProducts productId={product.state.data.id} />
        </div>
      )
  }
}

/** GET /api/v1/products/:id/related: popular products from the same category. */
function RelatedProducts({ productId }: { productId: string }) {
  const related = useRelatedProducts(productId)

  // ProductResults renders a page of results; wrap the plain list as one page.
  const state: AsyncState<Paginated<ProductSummary>> =
    related.state.status === 'success'
      ? {
          status: 'success',
          data: {
            items: related.state.data,
            page: 1,
            pageSize: related.state.data.length,
            total: related.state.data.length,
            totalPages: 1,
          },
        }
      : related.state

  return (
    <section aria-labelledby="related-products">
      <h2 id="related-products" className="mb-3 text-xl font-semibold">
        Produk terkait
      </h2>
      <ProductResults
        state={state}
        onRetry={related.reload}
        skeletonCount={4}
        emptyTitle="Belum ada produk terkait"
        emptyDescription="Belum ada produk lain di kategori ini."
      />
    </section>
  )
}
