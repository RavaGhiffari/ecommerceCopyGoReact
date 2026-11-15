// Seller API. The store is always the caller's own: the backend takes it from
// the session, so no store ID is ever sent from here.
import { apiRequest } from '../lib/api'
import type { Money } from '../types/marketplace'
import type {
  CreateProductInput,
  OpenStoreInput,
  SellerProduct,
  SellerProductSummary,
  SellerStats,
  SellerStore,
  UpdateProductInput,
  VariantInput,
} from '../types/seller'

const enc = encodeURIComponent

export const openStore = (input: OpenStoreInput) =>
  apiRequest<SellerStore>('POST', '/api/v1/seller/store', { body: input })

export const getMyStore = (signal?: AbortSignal) =>
  apiRequest<SellerStore>('GET', '/api/v1/seller/store', { signal })

export const listMyProducts = (signal?: AbortSignal) =>
  apiRequest<SellerProductSummary[]>('GET', '/api/v1/seller/products', { signal })

export const getMyProduct = (id: string, signal?: AbortSignal) =>
  apiRequest<SellerProduct>('GET', `/api/v1/seller/products/${enc(id)}`, { signal })

export const createProduct = (input: CreateProductInput) =>
  apiRequest<SellerProduct>('POST', '/api/v1/seller/products', { body: input })

export const updateProduct = (id: string, input: UpdateProductInput) =>
  apiRequest<SellerProduct>('PATCH', `/api/v1/seller/products/${enc(id)}`, { body: input })

export const archiveProduct = (id: string) =>
  apiRequest<void>('DELETE', `/api/v1/seller/products/${enc(id)}`)

export const addVariant = (id: string, input: VariantInput) =>
  apiRequest<SellerProduct>('POST', `/api/v1/seller/products/${enc(id)}/variants`, { body: input })

export const updateVariant = (id: string, variantId: string, input: { name?: string; price?: Money }) =>
  apiRequest<SellerProduct>('PATCH', `/api/v1/seller/products/${enc(id)}/variants/${enc(variantId)}`, { body: input })

export const setStock = (id: string, variantId: string, quantity: number, reason = '') =>
  apiRequest<{ variantId: string; availableQuantity: number }>(
    'PUT',
    `/api/v1/seller/products/${enc(id)}/variants/${enc(variantId)}/stock`,
    { body: { quantity, reason } },
  )

/** Dashboard numbers of the caller's store (6c). */
export const getMyStats = (signal?: AbortSignal) =>
  apiRequest<SellerStats>('GET', '/api/v1/seller/stats', { signal })
