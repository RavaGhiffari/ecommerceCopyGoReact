// Mirror of backend/internal/modules/orders/dto.go. Keep the two in sync.
// Money is a decimal string ("349000.00"); times are RFC 3339 UTC strings.
import type { Money, ProductImage } from './marketplace'

export type OrderStatus =
  | 'pending_payment'
  | 'paid'
  | 'processing'
  | 'shipped'
  | 'delivered'
  | 'cancelled'
  | 'refunded'

export type ShippingMethod = 'regular' | 'express' | 'same_day'

export interface OrderItem {
  id: string
  variantId: string
  productId: string
  productSlug: string
  productName: string // snapshot taken at checkout
  variantName: string
  sku: string
  unitPrice: Money // snapshot taken at checkout
  quantity: number
  lineTotal: Money
  image: ProductImage | null
}

export interface OrderStoreRef {
  id: string
  slug: string
  name: string
}

export interface ShippingAddress {
  recipientName: string
  phone: string
  addressLine: string
  city: string
  province: string
  postalCode: string
}

export interface OrderHistoryEntry {
  fromStatus: OrderStatus | null // null for the first entry
  toStatus: OrderStatus
  note: string
  createdAt: string
}

export interface Order {
  id: string
  checkoutId: string
  orderNumber: string // PK-YYYYMMDD-XXXXXX
  status: OrderStatus
  store: OrderStoreRef
  shippingMethod: ShippingMethod
  shippingFee: Money
  itemsSubtotal: Money
  total: Money
  shippingAddress: ShippingAddress
  items: OrderItem[]
  history?: OrderHistoryEntry[] // only on the detail endpoints
  createdAt: string
  updatedAt: string
}

export interface OrderPage {
  items: Order[]
  page: number
  pageSize: number
  total: number
  totalPages: number
}

/** Body of PATCH /api/v1/seller/orders/:id/status. */
export interface UpdateOrderStatusInput {
  status: OrderStatus
  note?: string
}

/** Mirrors orders.MaxNoteLength. */
export const MAX_NOTE_LENGTH = 500
