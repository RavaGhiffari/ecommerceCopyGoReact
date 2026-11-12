import { Link, useParams } from 'react-router-dom'
import { Pagination } from '../components/shared/Pagination'
import { EmptyState } from '../components/shared/StateViews'
import { useCategories, useProducts } from '../features/catalog/hooks'
import { PAGE_SIZE } from '../features/catalog/query'
import { ProductResults } from '../features/catalog/ProductResults'
import { SortSelect } from '../features/catalog/SortSelect'
import { useListParams } from '../features/catalog/useListParams'
import { usePageTitle } from '../hooks/usePageTitle'

/** Serves /products (all, with ?q= search) and /categories/:categorySlug. */
export function ProductListPage() {
  const { categorySlug } = useParams()
  const { q, sort, page, setSort, setPage } = useListParams()

  const categories = useCategories().state
  const products = useProducts({ q, categorySlug, sort, page, pageSize: PAGE_SIZE })

  const category =
    categorySlug && categories.status === 'success'
      ? categories.data.find((c) => c.slug === categorySlug)
      : undefined
  const unknownCategory = Boolean(categorySlug) && categories.status === 'success' && !category

  const heading = q ? `Hasil pencarian "${q}"` : (category?.name ?? 'Semua Produk')
  usePageTitle(unknownCategory ? 'Kategori tidak ditemukan' : heading)

  if (unknownCategory) {
    return (
      <EmptyState
        title="Kategori tidak ditemukan"
        description="Kategori ini tidak ada. Pilih kategori lain dari menu di atas."
        action={
          <Link to="/products" className="text-sm font-medium text-emerald-700 hover:underline">
            Lihat semua produk
          </Link>
        }
      />
    )
  }

  const countText =
    products.state.status === 'success'
      ? `${products.state.data.total} produk${q && category ? ` di ${category.name}` : ''}`
      : products.state.status === 'loading'
        ? 'Memuat…'
        : ''

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">{heading}</h1>
          <p className="min-h-5 text-sm text-slate-600" aria-live="polite">
            {countText}
          </p>
        </div>
        <SortSelect value={sort} onChange={setSort} />
      </div>

      <ProductResults
        state={products.state}
        onRetry={products.reload}
        skeletonCount={PAGE_SIZE}
        emptyTitle={q ? `Tidak ada hasil untuk "${q}"` : 'Belum ada produk'}
        emptyDescription={q ? 'Periksa ejaan atau coba kata kunci yang lebih umum.' : 'Produk akan muncul di sini.'}
        emptyAction={
          q ? (
            <Link to="/products" className="text-sm font-medium text-emerald-700 hover:underline">
              Hapus pencarian
            </Link>
          ) : undefined
        }
      />

      {products.state.status === 'success' && (
        <Pagination
          page={products.state.data.page}
          totalPages={products.state.data.totalPages}
          onPageChange={setPage}
        />
      )}
    </div>
  )
}
