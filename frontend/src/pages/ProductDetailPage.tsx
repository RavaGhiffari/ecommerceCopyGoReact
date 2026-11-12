import { Link, useParams } from 'react-router-dom'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { useProduct, useProducts } from '../features/catalog/hooks'
import { ProductDetailView } from '../features/catalog/ProductDetailView'
import { ProductResults } from '../features/catalog/ProductResults'
import type { AsyncState } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import type { Paginated, ProductDetail, ProductSummary } from '../types/marketplace'

export function ProductDetailPage() {
  const { storeSlug = '', productSlug = '' } = useParams()
  const product = useProduct(storeSlug, productSlug)

  usePageTitle(product.state.status === 'success' ? product.state.data.name : 'Produk')

  switch (product.state.status) {
    case 'loading':
      return (
        <div className="grid animate-pulse gap-8 md:grid-cols-2" role="status" aria-label="Memuat produk">
          <div className="aspect-square rounded-xl bg-slate-200" />
          <div className="space-y-4">
            <div className="h-8 w-3/4 rounded bg-slate-200" />
            <div className="h-5 w-1/3 rounded bg-slate-200" />
            <div className="h-10 w-1/2 rounded bg-slate-200" />
            <div className="h-24 rounded bg-slate-200" />
          </div>
        </div>
      )
    case 'error':
      return product.state.error.status === 404 ? (
        <EmptyState
          title="Produk tidak ditemukan"
          description="Produk ini tidak ada, sudah diarsipkan, atau alamatnya salah."
          action={
            <Link to="/products" className="text-sm font-medium text-emerald-700 hover:underline">
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
          <RelatedProducts product={product.state.data} />
        </div>
      )
  }
}

function RelatedProducts({ product }: { product: ProductDetail }) {
  const related = useProducts({
    categorySlug: product.category.slug,
    sort: 'popular',
    page: 1,
    pageSize: 5,
  })

  // Same category, minus the product being viewed, at most four.
  const state: AsyncState<Paginated<ProductSummary>> =
    related.state.status === 'success'
      ? {
          status: 'success',
          data: {
            ...related.state.data,
            items: related.state.data.items.filter((p) => p.id !== product.id).slice(0, 4),
          },
        }
      : related.state

  return (
    <section aria-labelledby="related-products">
      <h2 id="related-products" className="mb-3 text-xl font-semibold text-slate-900">
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
