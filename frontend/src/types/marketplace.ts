// Mirrors of the catalog API (backend/internal/modules/{categories,stores,products}/dto.go).
// Keep the two in sync.
//
// Money is a decimal STRING ("349000.00"), exactly as the backend stores it
// (NUMERIC). Only convert to a number at the last moment, for display.

/** A decimal amount of rupiah, e.g. "349000.00". */
export type Money = string

export interface Category {
  id: string
  slug: string
  name: string
  parentId: string | null // null = top-level
  sortOrder: number
}

export interface Rating {
  average: number // 0 when there are no reviews yet (reviews arrive in a later phase)
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
  price: Money // variant price is authoritative (CLAUDE.md §7)
  stock: number // available quantity
}

/** One row of a product listing. */
export interface ProductSummary {
  id: string
  slug: string // unique per store, not globally
  name: string
  categoryId: string
  images: ProductImage[] // first image only in listings
  lowestPrice: Money
  highestPrice: Money
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

export type ProductSort = 'popular' | 'newest' | 'price_asc' | 'price_desc'

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
