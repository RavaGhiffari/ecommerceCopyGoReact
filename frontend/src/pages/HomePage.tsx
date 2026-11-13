import { Link } from 'react-router-dom'
import { Boxes, Dumbbell, Home, Laptop, Shirt, ShoppingBag, type LucideIcon } from 'lucide-react'
import { useCategories } from '../features/catalog/hooks'
import { ProductSection } from '../features/catalog/ProductSection'
import { usePageTitle } from '../hooks/usePageTitle'

// Presentation only: the icon per category slug. Unknown slugs get a default.
const CATEGORY_ICONS: Record<string, LucideIcon> = {
  elektronik: Laptop,
  'fashion-pria': Shirt,
  'fashion-wanita': ShoppingBag,
  'rumah-tangga': Home,
  olahraga: Dumbbell,
}

export function HomePage() {
  usePageTitle()
  const { state: categories } = useCategories()

  return (
    <div className="space-y-10">
      <section className="rounded-2xl bg-gradient-to-r from-emerald-600 to-teal-500 px-6 py-10 text-white sm:px-10 sm:py-14">
        <p className="text-sm font-medium text-emerald-100">Marketplace multi-penjual</p>
        <h1 className="mt-2 max-w-xl text-3xl font-bold leading-tight sm:text-4xl">
          Belanja dari banyak toko, dalam satu tempat.
        </h1>
        <p className="mt-3 max-w-xl text-emerald-50">
          Jelajahi produk dari berbagai penjual. Keranjang dan checkout segera hadir.
        </p>
        <Link
          to="/products"
          className="mt-6 inline-block rounded-md bg-white px-5 py-2.5 text-sm font-semibold text-emerald-700 hover:bg-emerald-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-emerald-600"
        >
          Lihat semua produk
        </Link>
      </section>

      <section aria-labelledby="home-categories">
        <h2 id="home-categories" className="mb-3 text-xl font-semibold text-slate-900">
          Kategori
        </h2>
        {categories.status === 'success' ? (
          <ul className="grid grid-cols-3 gap-3 sm:grid-cols-6">
            {categories.data.filter((c) => c.parentId === null).map((c) => {
              const Icon = CATEGORY_ICONS[c.slug] ?? Boxes
              return (
                <li key={c.id}>
                  <Link
                    to={`/categories/${c.slug}`}
                    className="flex h-full flex-col items-center gap-2 rounded-xl border border-slate-200 bg-white p-3 text-center text-sm text-slate-700 hover:border-emerald-400 hover:text-emerald-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
                  >
                    <span className="rounded-full bg-emerald-50 p-3 text-emerald-600">
                      <Icon className="size-5" aria-hidden="true" />
                    </span>
                    {c.name}
                  </Link>
                </li>
              )
            })}
          </ul>
        ) : (
          <div className="grid animate-pulse grid-cols-3 gap-3 sm:grid-cols-6" role="status" aria-label="Memuat kategori">
            {Array.from({ length: 6 }, (_, i) => (
              <div key={i} className="h-24 rounded-xl bg-slate-200" />
            ))}
          </div>
        )}
      </section>

      <ProductSection
        title="Produk Terlaris"
        query={{ sort: 'popular', page: 1, pageSize: 4 }}
        seeAllTo="/products?sort=popular"
      />
      <ProductSection
        title="Harga Hemat"
        query={{ sort: 'price_asc', page: 1, pageSize: 4 }}
        seeAllTo="/products?sort=price_asc"
      />
      <ProductSection
        title="Baru Masuk"
        query={{ sort: 'newest', page: 1, pageSize: 4 }}
        seeAllTo="/products?sort=newest"
      />
    </div>
  )
}
