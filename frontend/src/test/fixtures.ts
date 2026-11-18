// Small builders for component tests. Each returns a complete, valid object
// that a test overrides only where it matters.
import type { User } from '../types/auth'
import type { Cart, CartLine } from '../types/cart'
import type { ProductSummary } from '../types/marketplace'
import type { AuthContextValue } from '../features/auth/authContext'
import type { CartContextValue } from '../features/cart/cartContext'

export function aUser(overrides: Partial<User> = {}): User {
  return {
    id: 'u-1',
    email: 'agus@contoh.id',
    name: 'Agus Wijaya',
    phone: null,
    status: 'active',
    roles: ['customer'],
    createdAt: '2026-11-01T00:00:00Z',
    ...overrides,
  }
}

const reject = () => Promise.reject(new Error('not used in this test'))

export function anAuth(overrides: Partial<AuthContextValue> = {}): AuthContextValue {
  return {
    status: 'anonymous',
    user: null,
    login: reject,
    register: reject,
    logout: reject,
    updateProfile: reject,
    refresh: () => {},
    ...overrides,
  }
}

export function signedIn(user: User = aUser()): AuthContextValue {
  return anAuth({ status: 'authenticated', user })
}

export function aProduct(overrides: Partial<ProductSummary> = {}): ProductSummary {
  return {
    id: 'p-1',
    slug: 'kabel-usb-c-braided',
    name: 'Kabel USB-C Braided',
    categoryId: 'c-1',
    images: [],
    lowestPrice: '39000.00',
    highestPrice: '39000.00',
    rating: { average: 4.5, count: 2 },
    soldCount: 21000,
    inStock: true,
    createdAt: '2026-11-01T00:00:00Z',
    store: { id: 's-1', slug: 'gadget-nusantara', name: 'Gadget Nusantara', location: 'Jakarta Barat' },
    ...overrides,
  }
}

export function aCartLine(overrides: Partial<CartLine> = {}): CartLine {
  return {
    id: 'ci-1',
    quantity: 2,
    variantId: 'v-1',
    variantName: '2 meter',
    sku: 'GN-KBL-2M',
    productId: 'p-1',
    productSlug: 'kabel-usb-c-braided',
    productName: 'Kabel USB-C Braided',
    image: null,
    unitPrice: '59000.00',
    lineTotal: '118000.00',
    availableQuantity: 50,
    issue: null,
    ...overrides,
  }
}

/** One store group holding `lines`; no lines = an empty cart (no groups, like the API). */
export function aCart(lines: CartLine[], subtotal: string): Cart {
  return {
    stores: lines.length === 0 ? [] : [
      {
        store: { id: 's-1', slug: 'gadget-nusantara', name: 'Gadget Nusantara', location: 'Jakarta Barat' },
        items: lines,
        subtotal,
      },
    ],
    itemCount: lines.reduce((n, l) => n + l.quantity, 0),
    subtotal,
    hasIssues: lines.some((l) => l.issue !== null),
  }
}

export function aCartContext(overrides: Partial<CartContextValue> = {}): CartContextValue {
  return {
    status: 'ready',
    cart: null,
    error: null,
    itemCount: 0,
    add: reject,
    update: reject,
    remove: reject,
    reload: () => {},
    ...overrides,
  }
}

/** Intl uses a non-breaking space in "Rp 39.000"; compare with plain spaces. */
export const plain = (s: string | null | undefined) => (s ?? '').replace(/\s/g, ' ')
