// Mirror of backend/internal/modules/checkout/dto.go. Keep the two in sync.
import type { Address } from './address'
import type { Money, ProductImage, StoreSummary } from './marketplace'
import type { Order, ShippingMethod } from './orders'

export type { ShippingMethod }

/** Body of POST /checkout/quote and POST /checkout: store id -> method. */
export interface CheckoutInput {
  addressId: string
  shipping: Record<string, ShippingMethod>
}

export type CheckoutLineIssue = 'unavailable' | 'out_of_stock' | 'insufficient_stock' | 'own_product'

export interface CheckoutLine {
  itemId: string // cart item id
  variantId: string
  variantName: string
  sku: string
  productId: string
  productSlug: string
  productName: string
  image: ProductImage | null
  unitPrice: Money
  quantity: number
  lineTotal: Money
  availableQuantity: number
  issue: CheckoutLineIssue | null
}

export interface ShippingOption {
  method: ShippingMethod
  fee: Money
}

export interface CheckoutGroup {
  store: StoreSummary
  items: CheckoutLine[]
  shippingOptions: ShippingOption[] // only the methods allowed for this address
  shippingMethod: ShippingMethod
  shippingFee: Money
  subtotal: Money
  total: Money
}

export interface Quote {
  address: Address
  stores: CheckoutGroup[]
  itemsTotal: Money
  shippingTotal: Money
  grandTotal: Money
  hasIssues: boolean
}

export type CheckoutStatus = 'pending_payment' | 'paid' | 'cancelled' | 'expired'
export type PaymentStatus = 'pending' | 'succeeded' | 'failed' | 'cancelled' | 'expired'

export interface Payment {
  id: string
  reference: string
  provider: string
  amount: Money
  status: PaymentStatus
  expiresAt: string
  paidAt: string | null
}

export interface Checkout {
  id: string
  status: CheckoutStatus
  itemsTotal: Money
  shippingTotal: Money
  grandTotal: Money
  expiresAt: string
  paidAt: string | null
  cancelledAt: string | null
  createdAt: string
  payment: Payment
  orders: Order[] // one per store
}

/** One entry of the 409 CART_HAS_ISSUES details ({ items: [...] }). */
export interface CheckoutIssue {
  itemId: string
  variantId: string
  productName: string
  variantName: string
  quantity: number
  availableQuantity: number
  issue: CheckoutLineIssue
}

export type PaymentOutcome = 'success' | 'failure'
