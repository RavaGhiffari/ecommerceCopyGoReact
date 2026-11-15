// Mirrors of the seller API (backend/internal/modules/{stores,products}/dto.go).
// Keep the two in sync.
import type { Money, ProductImage, ProductVariant, Rating } from './marketplace'
import type { OrderStatus } from './orders'

export interface SellerStore {
  id: string
  slug: string
  name: string
  description: string
  location: string
  status: 'active' | 'suspended'
  createdAt: string
}

export type ProductStatus = 'active' | 'archived'

export interface SellerProductSummary {
  id: string
  slug: string
  name: string
  status: ProductStatus
  categoryId: string
  variantCount: number
  totalStock: number
  lowestPrice: Money
  highestPrice: Money
  updatedAt: string
}

export interface SellerProduct {
  id: string
  slug: string
  name: string
  description: string
  status: ProductStatus
  categoryId: string
  images: ProductImage[]
  variants: ProductVariant[]
  createdAt: string
  updatedAt: string
}

export interface OpenStoreInput {
  name: string
  description: string
  location: string
}

export interface VariantInput {
  name: string
  sku: string
  price: Money // decimal string, sent as-is so no precision is lost
  stock: number
}

export interface CreateProductInput {
  name: string
  description: string
  categoryId: string
  images: ProductImage[]
  variants: VariantInput[]
}

export interface UpdateProductInput {
  name?: string
  description?: string
  categoryId?: string
  status?: ProductStatus
  images?: ProductImage[]
}

// ---- GET /seller/stats (6c), mirror of orders.SellerStatsResponse --------------

export interface SellerTopProduct {
  productId: string
  name: string
  status: ProductStatus
  units: number
  revenue: Money // items only, last 30 days
}

export interface SellerStats {
  /** Every order status as a key, 0 when the store has none. */
  ordersByStatus: Record<OrderStatus, number>
  /** Sum of items (no shipping) of paid, processing, shipped and delivered orders. */
  revenue: { last30Days: Money; allTime: Money }
  unitsSold: { last30Days: number; allTime: number }
  /** Last 30 days, at most 5, by units. */
  topProducts: SellerTopProduct[]
  rating: Rating
  /** Start of the 30-day window (RFC 3339). */
  since: string
}
