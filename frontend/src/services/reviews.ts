// Review API (Phase 6a). The author is always the session user; the server
// checks that the order item is the caller's and that the order is delivered.
import { apiRequest } from '../lib/api'
import type { Paginated } from '../types/marketplace'
import type {
  CreateReviewInput,
  MyReview,
  PendingReviewItem,
  ProductReviews,
  Review,
  UpdateReviewInput,
} from '../types/reviews'

const enc = encodeURIComponent

const paging = (page: number, pageSize: number) => `?page=${page}&pageSize=${pageSize}`

/** Public: rating summary + one page of reviews of a product. */
export const listProductReviews = (productId: string, page: number, pageSize: number, signal?: AbortSignal) =>
  apiRequest<ProductReviews>('GET', `/api/v1/products/${enc(productId)}/reviews${paging(page, pageSize)}`, { signal })

export const listMyReviews = (page: number, pageSize: number, signal?: AbortSignal) =>
  apiRequest<Paginated<MyReview>>('GET', `/api/v1/me/reviews${paging(page, pageSize)}`, { signal })

/** Delivered items of the caller that have no review yet. */
export const listPendingReviews = (page: number, pageSize: number, signal?: AbortSignal) =>
  apiRequest<Paginated<PendingReviewItem>>('GET', `/api/v1/me/reviews/pending${paging(page, pageSize)}`, { signal })

export const createReview = (input: CreateReviewInput) => apiRequest<Review>('POST', '/api/v1/reviews', { body: input })

export const updateReview = (id: string, input: UpdateReviewInput) =>
  apiRequest<Review>('PATCH', `/api/v1/reviews/${enc(id)}`, { body: input })
