// Shapes the real catalog API is expected to return (see CLAUDE.md §8/§9).
// During the UI prototype these come from src/services, backed by src/mocks.
// When the backend lands, only src/services changes; keep this file aligned
// with the real response contracts.
//
// Money: the backend stores NUMERIC. The prototype uses whole rupiah as a
// number; whether the API sends numbers or decimal strings is decided in the
// catalog phase.

export interface Category {
  id: string
  slug: string
  name: string
  parentId: string | null
}

export interface Rating {
  average: number // 0 when there are no reviews yet
  count: number
}

export interface ProductImage {
  url: string
  alt: string
}

/** The store fields shown next to a product (product cards, detail page). */
export interface StoreSummary {
  id: string
  slug: string
  name: string
  location: string
}

export interface StoreDetail extends StoreSummary {
  description: string
  rating: Rating
  joinedAt: string // ISO 8601
  activeProductCount: number
}

export interface ProductVariant {
  id: string
  sku: string
  name: string
  price: number // variant price is authoritative (CLAUDE.md §7)
  stock: number // available quantity
}

/** One row of a product listing. */
export interface ProductSummary {
  id: string
  slug: string // unique per store, not globally
  name: string
  categoryId: string
  images: ProductImage[]
  lowestPrice: number
  highestPrice: number
  rating: Rating
  soldCount: number
  inStock: boolean
  createdAt: string // ISO 8601
  store: StoreSummary
}

export interface ProductDetail extends ProductSummary {
  description: string
  variants: ProductVariant[]
  category: Category
}

export type ProductSort = 'popular' | 'newest' | 'rating' | 'price_asc' | 'price_desc'

export interface ProductQuery {
  q?: string
  categorySlug?: string
  storeSlug?: string
  sort: ProductSort
  page: number
  pageSize: number
}

export interface Paginated<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
  totalPages: number
}
