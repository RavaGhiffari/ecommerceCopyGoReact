// Mirror of backend/internal/modules/reviews/dto.go. Keep the two in sync.
// Times are RFC 3339 UTC strings.
import type { Paginated, ProductImage } from './marketplace'

export type ReviewStatus = 'visible' | 'hidden'

export interface Review {
  id: string
  orderItemId: string
  productId: string
  rating: number
  comment: string
  status: ReviewStatus // hidden = removed by an admin; only the author still sees it
  createdAt: string
  updatedAt: string
}

/** One row of a product's public review list (no email, no user id). */
export interface PublicReview {
  id: string
  rating: number
  comment: string
  reviewerName: string // already masked by the server, e.g. "Agus W***"
  variantName: string
  edited: boolean
  createdAt: string
}

export interface ReviewSummary {
  average: number // one decimal, 0 without reviews
  count: number
  distribution: Record<'1' | '2' | '3' | '4' | '5', number>
}

export interface ProductReviews extends Paginated<PublicReview> {
  summary: ReviewSummary
}

/** What a review is about: the order item, with links. Names are snapshots. */
export interface ReviewItemRef {
  orderItemId: string
  orderId: string
  orderNumber: string
  productId: string
  productSlug: string
  storeSlug: string
  productName: string
  variantName: string
  image: ProductImage | null
}

export interface MyReview extends Review {
  item: ReviewItemRef
}

export interface PendingReviewItem extends ReviewItemRef {
  storeName: string
  deliveredAt: string
}

export interface CreateReviewInput {
  orderItemId: string
  rating: number
  comment: string
}

export interface UpdateReviewInput {
  rating?: number
  comment?: string
}
