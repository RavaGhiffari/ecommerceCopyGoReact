import { createContext, useContext } from 'react'
import type { ApiError } from '../../lib/api'
import type { Cart } from '../../types/cart'

export type CartStatus = 'anonymous' | 'loading' | 'error' | 'ready'

export interface CartContextValue {
  status: CartStatus
  cart: Cart | null
  error: ApiError | null
  itemCount: number
  add: (variantId: string, quantity: number) => Promise<Cart>
  update: (itemId: string, quantity: number) => Promise<Cart>
  remove: (itemId: string) => Promise<Cart>
  reload: () => void
}

export const CartContext = createContext<CartContextValue | null>(null)

/** The signed-in user's cart, shared by the header badge, product page and /cart. */
export function useCart(): CartContextValue {
  const ctx = useContext(CartContext)
  if (!ctx) throw new Error('useCart must be used inside <CartProvider>')
  return ctx
}
