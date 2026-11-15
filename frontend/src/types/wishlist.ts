// Mirror of backend/internal/modules/wishlist/dto.go. Keep the two in sync.
import type { Money, Paginated, ProductImage } from './marketplace'

export interface WishlistItem {
  productId: string
  slug: string
  name: string
  image: ProductImage | null
  lowestPrice: Money // the cheapest variant, as a decimal string
  store: { slug: string; name: string }
  addedAt: string // RFC 3339
}

export type WishlistPage = Paginated<WishlistItem>
