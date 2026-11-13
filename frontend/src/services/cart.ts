// Cart API. The cart always belongs to the logged-in user (the server takes it
// from the session); every mutation returns the whole updated cart.
import { apiRequest } from '../lib/api'
import type { Cart } from '../types/cart'

const enc = encodeURIComponent

export const getCart = (signal?: AbortSignal) => apiRequest<Cart>('GET', '/api/v1/cart', { signal })

export const addCartItem = (variantId: string, quantity: number) =>
  apiRequest<Cart>('POST', '/api/v1/cart/items', { body: { variantId, quantity } })

export const updateCartItem = (itemId: string, quantity: number) =>
  apiRequest<Cart>('PATCH', `/api/v1/cart/items/${enc(itemId)}`, { body: { quantity } })

export const removeCartItem = (itemId: string) => apiRequest<Cart>('DELETE', `/api/v1/cart/items/${enc(itemId)}`)
