// Mirror of the admin API (Phase 6d):
//   backend/internal/modules/admin/handler.go       users, stores, stats, audit logs
//   backend/internal/modules/reviews/admin.go       AdminReviewResponse
//   backend/internal/modules/orders/admin.go        orders reuse types/orders.ts
// Keep them in sync. Money is a decimal string; times are RFC 3339 UTC.
import type { Role } from './auth'
import type { Money } from './marketplace'
import type { OrderStatus } from './orders'
import type { ReviewStatus } from './reviews'

export type AccountStatus = 'active' | 'suspended'

/** Every admin list answers with this page shape. */
export interface AdminPage<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
  totalPages: number
}

export interface AdminUser {
  id: string
  email: string
  name: string
  status: AccountStatus
  roles: Role[]
  createdAt: string
}

export interface AdminStore {
  id: string
  name: string
  slug: string
  location: string
  status: AccountStatus
  owner: { id: string; name: string; email: string }
  activeProductCount: number
  createdAt: string
}

export interface AdminStats {
  users: Partial<Record<AccountStatus, number>>
  stores: Partial<Record<AccountStatus, number>>
  publicProducts: number
  ordersByStatus: Partial<Record<OrderStatus, number>>
  gmv: { last30Days: Money; allTime: Money }
  since: string // start of the 30-day window
}

export interface AdminReview {
  id: string
  rating: number
  comment: string
  status: ReviewStatus
  edited: boolean
  product: { id: string; name: string } // name = order item snapshot
  storeName: string
  reviewer: { name: string; email: string }
  createdAt: string
}

export type AuditAction =
  | 'user.suspend'
  | 'user.reactivate'
  | 'store.suspend'
  | 'store.reactivate'
  | 'category.create'
  | 'category.update'
  | 'review.hide'
  | 'review.unhide'

export interface AuditLog {
  id: string
  action: AuditAction | string // unknown future actions still render
  targetType: 'user' | 'store' | 'category' | 'review' | string
  targetId: string
  details: Record<string, unknown>
  actor: { id: string; name: string; email: string }
  createdAt: string
}

/** Body of POST /admin/categories and PUT /admin/categories/:id. */
export interface CategoryInput {
  name: string
  slug: string // "" = derived from the name by the server
  parentId: string | null
  sortOrder: number
}
