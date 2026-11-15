import { Skeleton } from '@/components/ui/skeleton'
import { Link } from 'react-router-dom'
import { Boxes, Dumbbell, Home, Laptop, Shirt, ShoppingBag, type LucideIcon } from 'lucide-react'
import { useCategories } from '../features/catalog/hooks'
import { ProductSection } from '../features/catalog/ProductSection'
import { BannerCarousel, type Banner } from '../features/home/BannerCarousel'
import { usePageTitle } from '../hooks/usePageTitle'

// Presentation only: static promo banners (no promo backend exists). Every
// link points at a real route; nothing here pretends to be a real discount.
const BANNERS: Banner[] = [
  {
    id: 'welcome',
    eyebrow: 'Marketplace multi-penjual',
    title: 'Belanja dari banyak toko, dalam satu tempat.',
    description: 'Jelajahi produk dari berbagai penjual, checkout sekali untuk semua toko.',
    cta: { label: 'Lihat semua produk', to: '/products' },
    className: 'bg-gradient-to-r from-emerald-600 to-teal-500',
  },
  {
    id: 'popular',
    eyebrow: 'Paling diminati',
    title: 'Produk terlaris pilihan pembeli.',
    description: 'Lihat apa yang paling banyak dibeli minggu ini dari semua toko.',
    cta: { label: 'Lihat terlaris', to: '/products?sort=popular' },
    className: 'bg-gradient-to-r from-sky-600 to-indigo-500',
  },
  {
    id: 'electronics',
    eyebrow: 'Kategori unggulan',
    title: 'Gadget dan elektronik untuk setiap hari.',
    description: 'Dari aksesori kecil sampai perangkat utama, langsung dari penjualnya.',
    cta: { label: 'Jelajahi elektronik', to: '/categories/elektronik' },
    className: 'bg-gradient-to-r from-violet-600 to-fuchsia-500',
  },
  {
    id: 'sell',
    eyebrow: 'Untuk penjual',
    title: 'Buka tokomu sendiri, gratis.',
    description: 'Kelola produk, stok, dan pesanan dari satu dasbor penjual.',
    cta: { label: 'Mulai berjualan', to: '/seller' },
    className: 'bg-gradient-to-r from-amber-500 to-orange-500',
  },
]

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
      <BannerCarousel banners={BANNERS} intervalMs={10_000} />

      <section aria-labelledby="home-categories">
        <h2 id="home-categories" className="mb-3 text-xl font-semibold">
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
                    className="flex h-full flex-col items-center gap-2 rounded-xl bg-card p-3 text-center text-sm text-foreground/80 ring-1 ring-foreground/10 transition outline-none hover:text-primary hover:ring-primary/50 focus-visible:ring-3 focus-visible:ring-ring/50"
                  >
                    <span className="rounded-full bg-primary/10 p-3 text-primary">
                      <Icon className="size-5" aria-hidden="true" />
                    </span>
                    {c.name}
                  </Link>
                </li>
              )
            })}
          </ul>
        ) : (
          <div className="grid grid-cols-3 gap-3 sm:grid-cols-6" role="status" aria-label="Memuat kategori">
            {Array.from({ length: 6 }, (_, i) => (
              <Skeleton key={i} className="h-24 rounded-xl" />
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
