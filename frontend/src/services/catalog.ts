// Public catalog data, from the real API.
//
// This file used to read mock fixtures; the function signatures stayed the
// same, so no page or component changed when the backend arrived.
import { apiGet } from '../lib/api'
import type {
  Category,
  Paginated,
  ProductDetail,
  ProductQuery,
  ProductSummary,
  StoreDetail,
} from '../types/marketplace'

export function listCategories(signal?: AbortSignal): Promise<Category[]> {
  return apiGet<Category[]>('/api/v1/categories', signal)
}

export function listProducts(query: ProductQuery, signal?: AbortSignal): Promise<Paginated<ProductSummary>> {
  const params = new URLSearchParams({
    sort: query.sort,
    page: String(query.page),
    pageSize: String(query.pageSize),
  })
  if (query.q) params.set('q', query.q)
  if (query.categorySlug) params.set('category', query.categorySlug)
  if (query.storeSlug) params.set('store', query.storeSlug)
  return apiGet<Paginated<ProductSummary>>(`/api/v1/products?${params}`, signal)
}

export function getProduct(storeSlug: string, productSlug: string, signal?: AbortSignal): Promise<ProductDetail> {
  return apiGet<ProductDetail>(
    `/api/v1/stores/${encodeURIComponent(storeSlug)}/products/${encodeURIComponent(productSlug)}`,
    signal,
  )
}

export function getRelatedProducts(productId: string, signal?: AbortSignal): Promise<ProductSummary[]> {
  return apiGet<ProductSummary[]>(`/api/v1/products/${encodeURIComponent(productId)}/related`, signal)
}

export function getStore(slug: string, signal?: AbortSignal): Promise<StoreDetail> {
  return apiGet<StoreDetail>(`/api/v1/stores/${encodeURIComponent(slug)}`, signal)
}
