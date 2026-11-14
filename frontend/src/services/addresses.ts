// Address book API. Addresses always belong to the logged-in user (the server
// takes the owner from the session); someone else's id is a 404.
import { apiRequest } from '../lib/api'
import type { Address, AddressInput, AddressUpdate } from '../types/address'

const enc = encodeURIComponent
const BASE = '/api/v1/me/addresses'

export const listAddresses = (signal?: AbortSignal) => apiRequest<Address[]>('GET', BASE, { signal })

export const createAddress = (input: AddressInput) => apiRequest<Address>('POST', BASE, { body: input })

export const updateAddress = (id: string, input: AddressUpdate) =>
  apiRequest<Address>('PATCH', `${BASE}/${enc(id)}`, { body: input })

export const deleteAddress = (id: string) => apiRequest<void>('DELETE', `${BASE}/${enc(id)}`)

export const setDefaultAddress = (id: string) => apiRequest<Address>('POST', `${BASE}/${enc(id)}/default`)
