import { Link, NavLink, useParams } from 'react-router-dom'
import type { Category } from '../types/marketplace'
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
          <Link to="/products" className="text-sm font-medium text-primary hover:underline">
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
          <h1 className="text-2xl font-semibold">{heading}</h1>
          <p className="min-h-5 text-sm text-muted-foreground" aria-live="polite">
            {countText}
          </p>
        </div>
        <SortSelect value={sort} onChange={setSort} />
      </div>

      {category && categories.status === 'success' && (
        <SubcategoryChips current={category} all={categories.data} />
      )}

      <ProductResults
        state={products.state}
        onRetry={products.reload}
        skeletonCount={PAGE_SIZE}
        emptyTitle={q ? `Tidak ada hasil untuk "${q}"` : 'Belum ada produk'}
        emptyDescription={q ? 'Periksa ejaan atau coba kata kunci yang lebih umum.' : 'Produk akan muncul di sini.'}
        emptyAction={
          q ? (
            <Link to="/products" className="text-sm font-medium text-primary hover:underline">
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

/**
 * Two-level navigation: on "Elektronik" or any of its subcategories, show
 * "Semua Elektronik" plus every subcategory. Browsing a parent includes the
 * products of its subcategories (the API does that).
 */
function SubcategoryChips({ current, all }: { current: Category; all: Category[] }) {
  const parent = current.parentId ? all.find((c) => c.id === current.parentId) : current
  if (!parent) return null
  const children = all.filter((c) => c.parentId === parent.id)
  if (children.length === 0) return null

  const chip = ({ isActive }: { isActive: boolean }) =>
    `whitespace-nowrap rounded-full border px-3 py-1 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/50 ${
      isActive
        ? 'border-primary bg-primary/10 font-medium text-primary'
        : 'bg-card text-foreground/80 hover:border-foreground/30'
    }`

  return (
    <nav aria-label={`Subkategori ${parent.name}`} className="mb-5 flex gap-2 overflow-x-auto pb-1">
      <NavLink to={`/categories/${parent.slug}`} end className={chip}>
        Semua {parent.name}
      </NavLink>
      {children.map((c) => (
        <NavLink key={c.id} to={`/categories/${c.slug}`} end className={chip}>
          {c.name}
        </NavLink>
      ))}
    </nav>
  )
}
