// Catalog data access. During the UI prototype this is backed by src/mocks.
// It is the ONLY module allowed to import from src/mocks. When the backend
// exists, rewrite the bodies to call apiGet(...) and keep the signatures, so
// hooks and components do not change.
import { ApiError } from '../lib/api'
import { categories as categoryRecords } from '../mocks/categories'
import { products as productRecords, type ProductRecord } from '../mocks/products'
import { stores as storeRecords, type StoreRecord } from '../mocks/stores'
import type {
  Category,
  Paginated,
  ProductDetail,
  ProductQuery,
  ProductSummary,
  StoreDetail,
  StoreSummary,
} from '../types/marketplace'
import { mockRequest } from './simulate'

// ---- mapping: database-shaped rows -> API-shaped objects --------------------

const storeById = (id: string): StoreRecord => {
  const store = storeRecords.find((s) => s.id === id)
  if (!store) throw new Error(`mock data error: unknown store ${id}`)
  return store
}

const toStoreSummary = (s: StoreRecord): StoreSummary => ({
  id: s.id,
  slug: s.slug,
  name: s.name,
  location: s.location,
})

const toSummary = (p: ProductRecord): ProductSummary => {
  const prices = p.variants.map((v) => v.price)
  return {
    id: p.id,
    slug: p.slug,
    name: p.name,
    categoryId: p.categoryId,
    images: [], // the prototype has no photos; the UI draws a placeholder
    lowestPrice: Math.min(...prices),
    highestPrice: Math.max(...prices),
    rating: { average: p.ratingAverage, count: p.ratingCount },
    soldCount: p.soldCount,
    inStock: p.variants.some((v) => v.stock > 0),
    createdAt: p.createdAt,
    store: toStoreSummary(storeById(p.storeId)),
  }
}

const activeProducts = () => productRecords.filter((p) => p.status === 'active')

// ---- queries ----------------------------------------------------------------

export async function listCategories(): Promise<Category[]> {
  await mockRequest()
  return categoryRecords
}

export async function listProducts(query: ProductQuery): Promise<Paginated<ProductSummary>> {
  const sim = await mockRequest()

  let rows = sim === 'empty' ? [] : activeProducts()

  if (query.categorySlug) {
    const category = categoryRecords.find((c) => c.slug === query.categorySlug)
    rows = category ? rows.filter((p) => p.categoryId === category.id) : []
  }
  if (query.storeSlug) {
    const store = storeRecords.find((s) => s.slug === query.storeSlug)
    rows = store ? rows.filter((p) => p.storeId === store.id) : []
  }

  // Every word typed must appear in the name or description.
  const words = (query.q ?? '').toLowerCase().split(/\s+/).filter(Boolean)
  if (words.length > 0) {
    rows = rows.filter((p) => {
      const haystack = `${p.name} ${p.description}`.toLowerCase()
      return words.every((w) => haystack.includes(w))
    })
  }

  const summaries = rows.map(toSummary)
  summaries.sort(sorters[query.sort])

  const total = summaries.length
  const totalPages = Math.max(1, Math.ceil(total / query.pageSize))
  const page = Math.min(Math.max(1, query.page), totalPages)
  const start = (page - 1) * query.pageSize

  return {
    items: summaries.slice(start, start + query.pageSize),
    page,
    pageSize: query.pageSize,
    total,
    totalPages,
  }
}

const sorters: Record<ProductQuery['sort'], (a: ProductSummary, b: ProductSummary) => number> = {
  popular: (a, b) => b.soldCount - a.soldCount,
  newest: (a, b) => b.createdAt.localeCompare(a.createdAt),
  rating: (a, b) => b.rating.average - a.rating.average || b.rating.count - a.rating.count,
  price_asc: (a, b) => a.lowestPrice - b.lowestPrice,
  price_desc: (a, b) => b.lowestPrice - a.lowestPrice,
}

export async function getProduct(storeSlug: string, productSlug: string): Promise<ProductDetail> {
  const sim = await mockRequest()

  const store = storeRecords.find((s) => s.slug === storeSlug)
  const record = store
    ? activeProducts().find((p) => p.storeId === store.id && p.slug === productSlug)
    : undefined
  if (sim === 'empty' || !record) {
    throw new ApiError(404, 'PRODUCT_NOT_FOUND', 'Produk tidak ditemukan.')
  }

  const category = categoryRecords.find((c) => c.id === record.categoryId)
  if (!category) throw new Error(`mock data error: unknown category ${record.categoryId}`)

  return {
    ...toSummary(record),
    description: record.description,
    variants: record.variants.map((v) => ({ ...v })),
    category,
  }
}

export async function getStore(slug: string): Promise<StoreDetail> {
  const sim = await mockRequest()

  const store = storeRecords.find((s) => s.slug === slug)
  if (sim === 'empty' || !store) {
    throw new ApiError(404, 'STORE_NOT_FOUND', 'Toko tidak ditemukan.')
  }

  return {
    ...toStoreSummary(store),
    description: store.description,
    rating: { average: store.ratingAverage, count: store.ratingCount },
    joinedAt: store.joinedAt,
    activeProductCount: activeProducts().filter((p) => p.storeId === store.id).length,
  }
}
