import { useCallback, useMemo, useState, type ReactNode } from 'react'
import { useAsync } from '../../hooks/useAsync'
import { addCartItem, getCart, removeCartItem, updateCartItem } from '../../services/cart'
import type { Cart } from '../../types/cart'
import { useAuth } from '../auth/authContext'
import { CartContext, type CartContextValue } from './cartContext'

/**
 * Loads the cart whenever a user is signed in and keeps the latest version
 * returned by the server. Every mutation endpoint returns the whole cart, so
 * the UI never computes totals itself.
 */
// Every login (or profile refresh) yields a new user object. Numbering those
// objects makes the cart key change too, so a re-login always refetches the
// cart instead of showing a result kept from the previous session.
const sessionIds = new WeakMap<object, number>()
let nextSessionId = 0
function sessionIdOf(user: object): number {
  let id = sessionIds.get(user)
  if (id === undefined) {
    id = ++nextSessionId
    sessionIds.set(user, id)
  }
  return id
}

export function CartProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const key = user ? `cart:${user.id}:${sessionIdOf(user)}` : 'cart:anonymous'

  const loaded = useAsync<Cart | null>(() => (user ? getCart() : Promise.resolve(null)), key)
  // The cart returned by the last mutation, for the same user. A reload or a
  // different user discards it.
  const [latest, setLatest] = useState<{ key: string; cart: Cart } | null>(null)

  const keep = useCallback(
    async (request: Promise<Cart>) => {
      const cart = await request
      setLatest({ key, cart })
      return cart
    },
    [key],
  )

  const reloadLoaded = loaded.reload
  const value = useMemo<CartContextValue>(() => {
    const fromMutation = latest !== null && latest.key === key ? latest.cart : null
    let status: CartContextValue['status']
    let cart: Cart | null = fromMutation
    let error: CartContextValue['error'] = null

    if (!user) {
      status = 'anonymous'
      cart = null
    } else if (fromMutation) {
      status = 'ready'
    } else if (loaded.state.status === 'success') {
      status = 'ready'
      cart = loaded.state.data
    } else if (loaded.state.status === 'error') {
      status = 'error'
      error = loaded.state.error
    } else {
      status = 'loading'
    }

    return {
      status,
      cart,
      error,
      itemCount: cart?.itemCount ?? 0,
      add: (variantId, quantity) => keep(addCartItem(variantId, quantity)),
      update: (itemId, quantity) => keep(updateCartItem(itemId, quantity)),
      remove: (itemId) => keep(removeCartItem(itemId)),
      reload: () => {
        setLatest(null)
        reloadLoaded()
      },
    }
  }, [user, key, latest, loaded.state, keep, reloadLoaded])

  return <CartContext.Provider value={value}>{children}</CartContext.Provider>
}
