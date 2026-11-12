// Auth calls against the REAL backend (no mocks). The session lives in an
// HttpOnly cookie the browser manages; nothing here stores tokens.
import { ApiError, apiRequest } from '../lib/api'
import type { LoginInput, RegisterInput, UpdateProfileInput, User } from '../types/auth'

export function register(input: RegisterInput): Promise<User> {
  return apiRequest<User>('POST', '/api/v1/auth/register', { body: input })
}

export function login(input: LoginInput): Promise<User> {
  return apiRequest<User>('POST', '/api/v1/auth/login', { body: input })
}

export function logout(): Promise<void> {
  return apiRequest<void>('POST', '/api/v1/auth/logout')
}

/** The current user, or null when not logged in (401). */
export async function getMe(signal?: AbortSignal): Promise<User | null> {
  try {
    return await apiRequest<User>('GET', '/api/v1/me', { signal })
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return null
    throw err
  }
}

export function updateProfile(input: UpdateProfileInput): Promise<User> {
  return apiRequest<User>('PATCH', '/api/v1/me', { body: input })
}
