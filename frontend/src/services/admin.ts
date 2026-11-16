// Admin API (/api/v1/admin/*). Every route is behind RequireRole(admin) on
// the server; the UI guard (RequireAdmin) only saves a pointless round trip.
import { apiRequest } from '../lib/api'
import type {
  AccountStatus,
  AdminPage,
  AdminReview,
  AdminStats,
  AdminStore,
  AdminUser,
  AuditLog,
  CategoryInput,
} from '../types/admin'
import type { Category } from '../types/marketplace'
import type { Order, OrderPage, OrderStatus } from '../types/orders'
import type { Review, ReviewStatus } from '../types/reviews'

const enc = encodeURIComponent
const BASE = '/api/v1/admin'

function query(params: Record<string, string | number | undefined>): string {
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '') q.set(k, String(v))
  const s = q.toString()
  return s ? `?${s}` : ''
}

export interface ListFilter<S extends string> {
  q?: string
  status?: S
  page: number
}

export const getAdminStats = (signal?: AbortSignal) => apiRequest<AdminStats>('GET', `${BASE}/stats`, { signal })

export const listAdminUsers = (f: ListFilter<AccountStatus>, signal?: AbortSignal) =>
  apiRequest<AdminPage<AdminUser>>('GET', `${BASE}/users${query({ q: f.q, status: f.status, page: f.page })}`, {
    signal,
  })

export const setUserStatus = (id: string, status: AccountStatus) =>
  apiRequest<AdminUser>('PATCH', `${BASE}/users/${enc(id)}/status`, { body: { status } })

export const listAdminStores = (f: ListFilter<AccountStatus>, signal?: AbortSignal) =>
  apiRequest<AdminPage<AdminStore>>('GET', `${BASE}/stores${query({ q: f.q, status: f.status, page: f.page })}`, {
    signal,
  })

export const setStoreStatus = (id: string, status: AccountStatus) =>
  apiRequest<AdminStore>('PATCH', `${BASE}/stores/${enc(id)}/status`, { body: { status } })

export const createCategory = (input: CategoryInput) =>
  apiRequest<Category>('POST', `${BASE}/categories`, { body: input })

export const replaceCategory = (id: string, input: CategoryInput) =>
  apiRequest<Category>('PUT', `${BASE}/categories/${enc(id)}`, { body: input })

export const listAdminOrders = (status: OrderStatus | undefined, page: number, signal?: AbortSignal) =>
  apiRequest<OrderPage>('GET', `${BASE}/orders${query({ status, page })}`, { signal })

export const getAdminOrder = (id: string, signal?: AbortSignal) =>
  apiRequest<Order>('GET', `${BASE}/orders/${enc(id)}`, { signal })

export const listAdminReviews = (status: ReviewStatus | undefined, page: number, signal?: AbortSignal) =>
  apiRequest<AdminPage<AdminReview>>('GET', `${BASE}/reviews${query({ status, page })}`, { signal })

/** Answers with the public review shape; the page only needs its new status. */
export const setReviewStatus = (id: string, status: ReviewStatus) =>
  apiRequest<Review>('PATCH', `${BASE}/reviews/${enc(id)}/status`, { body: { status } })

export const listAuditLogs = (page: number, signal?: AbortSignal) =>
  apiRequest<AdminPage<AuditLog>>('GET', `${BASE}/audit-logs${query({ page })}`, { signal })
