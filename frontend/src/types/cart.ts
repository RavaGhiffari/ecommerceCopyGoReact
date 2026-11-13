// Mirror of backend/internal/modules/cart/dto.go. Keep the two in sync.
import type { Money, ProductImage, StoreSummary } from './marketplace'

export type CartIssue = 'unavailable' | 'out_of_stock' | 'insufficient_stock'

export interface CartLine {
  id: string
  quantity: number
  variantId: string
  variantName: string
  sku: string
  productId: string
  productSlug: string
  productName: string
  image: ProductImage | null
  unitPrice: Money // CURRENT price; orders will snapshot it later
  lineTotal: Money
  availableQuantity: number
  issue: CartIssue | null
}

export interface CartStoreGroup {
  store: StoreSummary
  items: CartLine[]
  subtotal: Money // lines without an issue only
}

export interface Cart {
  stores: CartStoreGroup[]
  itemCount: number // total units, for the header badge
  subtotal: Money // lines without an issue only
  hasIssues: boolean
}

export const MAX_QUANTITY_PER_LINE = 99
