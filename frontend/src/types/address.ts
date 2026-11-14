// Mirrors backend/internal/modules/addresses/dto.go. Keep both in sync.

export interface Address {
  id: string
  label: string
  recipientName: string
  phone: string // stored normalised: optional "+", then digits only
  addressLine: string
  city: string
  province: string
  postalCode: string
  isDefault: boolean
  createdAt: string
  updatedAt: string
}

/** Body of POST /api/v1/me/addresses. */
export interface AddressInput {
  label: string
  recipientName: string
  phone: string
  addressLine: string
  city: string
  province: string
  postalCode: string
  isDefault?: boolean
}

/** Body of PATCH /api/v1/me/addresses/:id. The default flag has its own route. */
export type AddressUpdate = Partial<Omit<AddressInput, 'isDefault'>>

/** Mirrors addresses.MaxPerUser. */
export const MAX_ADDRESSES = 10
