// Mirror of backend/internal/modules/users/dto.go (UserResponse).
// Keep the two in sync.

export type Role = 'customer' | 'seller' | 'admin'

export interface User {
  id: string
  email: string
  name: string
  phone: string | null
  status: 'active' | 'suspended'
  roles: Role[]
  createdAt: string // ISO 8601
}

export interface RegisterInput {
  name: string
  email: string
  password: string
}

export interface LoginInput {
  email: string
  password: string
}

/** PATCH /api/v1/me: omitted fields stay unchanged, phone "" clears it. */
export interface UpdateProfileInput {
  name?: string
  phone?: string
}
