// Order API for buyers (/orders) and sellers (/seller/orders). The server
// takes the buyer, or the seller's store, from the session.
import { apiRequest } from '../lib/api'
import type { Order, OrderPage, OrderStatus, UpdateOrderStatusInput } from '../types/orders'

const enc = encodeURIComponent

function query(params: Record<string, string | number | undefined>): string {
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '') q.set(k, String(v))
  const s = q.toString()
  return s ? `?${s}` : ''
}

export const listMyOrders = (page: number, signal?: AbortSignal) =>
  apiRequest<OrderPage>('GET', `/api/v1/orders${query({ page })}`, { signal })

export const getMyOrder = (id: string, signal?: AbortSignal) =>
  apiRequest<Order>('GET', `/api/v1/orders/${enc(id)}`, { signal })

/** Cancels a paid order before the seller processes it (simulated refund). */
export const cancelMyOrder = (id: string) => apiRequest<Order>('POST', `/api/v1/orders/${enc(id)}/cancel`)

export const listStoreOrders = (status: OrderStatus | undefined, page: number, signal?: AbortSignal) =>
  apiRequest<OrderPage>('GET', `/api/v1/seller/orders${query({ status, page })}`, { signal })

export const getStoreOrder = (id: string, signal?: AbortSignal) =>
  apiRequest<Order>('GET', `/api/v1/seller/orders/${enc(id)}`, { signal })

export const updateStoreOrderStatus = (id: string, input: UpdateOrderStatusInput) =>
  apiRequest<Order>('PATCH', `/api/v1/seller/orders/${enc(id)}/status`, { body: input })
