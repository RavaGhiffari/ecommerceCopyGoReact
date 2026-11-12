import { Link, useParams } from 'react-router-dom'
import { CalendarDays, MapPin } from 'lucide-react'
import { Pagination } from '../components/shared/Pagination'
import { RatingBadge } from '../components/shared/RatingBadge'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { useProducts, useStore } from '../features/catalog/hooks'
import { PAGE_SIZE } from '../features/catalog/query'
import { ProductResults } from '../features/catalog/ProductResults'
import { SortSelect } from '../features/catalog/SortSelect'
import { useListParams } from '../features/catalog/useListParams'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatMonthYear } from '../lib/format'

export function StorePage() {
  const { storeSlug = '' } = useParams()
  const { sort, page, setSort, setPage } = useListParams()

  const store = useStore(storeSlug)
  const products = useProducts({ storeSlug, sort, page, pageSize: PAGE_SIZE })

  usePageTitle(store.state.status === 'success' ? store.state.data.name : 'Toko')

  if (store.state.status === 'error') {
    return store.state.error.status === 404 ? (
      <EmptyState
        title="Toko tidak ditemukan"
        description="Toko ini tidak ada atau sudah tidak aktif."
        action={
          <Link to="/products" className="text-sm font-medium text-emerald-700 hover:underline">
            Lihat semua produk
          </Link>
        }
      />
    ) : (
      <ErrorState error={store.state.error} onRetry={store.reload} title="Gagal memuat toko" />
    )
  }

  return (
    <div className="space-y-6">
      {store.state.status === 'loading' ? (
        <div className="h-36 animate-pulse rounded-xl bg-slate-200" role="status" aria-label="Memuat toko" />
      ) : (
        <section className="rounded-xl border border-slate-200 bg-white p-5">
          <div className="flex items-start gap-4">
            <div
              aria-hidden="true"
              className="flex size-16 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-2xl font-bold text-emerald-700"
            >
              {store.state.data.name[0]}
            </div>
            <div className="min-w-0">
              <h1 className="text-2xl font-semibold text-slate-900">{store.state.data.name}</h1>
              <p className="mt-1 text-sm text-slate-600">{store.state.data.description}</p>
              <div className="mt-3 flex flex-wrap items-center gap-x-5 gap-y-1 text-sm text-slate-600">
                <span className="inline-flex items-center gap-1">
                  <MapPin className="size-4" aria-hidden="true" /> {store.state.data.location}
                </span>
                <RatingBadge rating={store.state.data.rating} />
                <span className="inline-flex items-center gap-1">
                  <CalendarDays className="size-4" aria-hidden="true" /> Bergabung{' '}
                  {formatMonthYear(store.state.data.joinedAt)}
                </span>
                <span>{store.state.data.activeProductCount} produk aktif</span>
              </div>
            </div>
          </div>
        </section>
      )}

      <section aria-labelledby="store-products">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <h2 id="store-products" className="text-xl font-semibold text-slate-900">
            Produk
          </h2>
          <SortSelect value={sort} onChange={setSort} />
        </div>
        <ProductResults
          state={products.state}
          onRetry={products.reload}
          skeletonCount={PAGE_SIZE}
          emptyTitle="Toko ini belum punya produk"
        />
        {products.state.status === 'success' && (
          <Pagination
            page={products.state.data.page}
            totalPages={products.state.data.totalPages}
            onPageChange={setPage}
          />
        )}
      </section>
    </div>
  )
}
