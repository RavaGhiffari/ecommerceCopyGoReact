// Wishlist API (practice feature). The owner is always the session user.
// Add and remove are idempotent (PUT / DELETE, 204 No Content).
import { apiRequest } from '../lib/api'
import type { WishlistPage } from '../types/wishlist'

const enc = encodeURIComponent

export const listWishlist = (page: number, pageSize: number, signal?: AbortSignal) =>
  apiRequest<WishlistPage>('GET', `/api/v1/me/wishlist?page=${page}&pageSize=${pageSize}`, { signal })

/** Ids of the saved products that are still public (for the heart buttons). */
export const listWishlistIds = (signal?: AbortSignal) =>
  apiRequest<{ productIds: string[] }>('GET', '/api/v1/me/wishlist/ids', { signal })

export const addToWishlist = (productId: string) =>
  apiRequest<void>('PUT', `/api/v1/me/wishlist/${enc(productId)}`)

export const removeFromWishlist = (productId: string) =>
  apiRequest<void>('DELETE', `/api/v1/me/wishlist/${enc(productId)}`)
