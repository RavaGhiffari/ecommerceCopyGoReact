import type { ApiEnvelope } from '../types/api'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? ''

/**
 * Every failed API call becomes an ApiError, whether the server answered with
 * our error envelope, answered with something unexpected, or was unreachable.
 * UI code then only has to handle one error type.
 */
export class ApiError extends Error {
  readonly status: number // HTTP status; 0 when the server was unreachable
  readonly code: string // backend error code, e.g. DATABASE_UNAVAILABLE
  readonly details?: unknown

  constructor(status: number, code: string, message: string, details?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
  }
}

function isEnvelope(body: unknown): body is ApiEnvelope<unknown> {
  return typeof body === 'object' && body !== null && 'success' in body
}

/** GET a JSON endpoint and return the envelope's `data`, or throw ApiError. */
export async function apiGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  let res: Response
  try {
    res = await fetch(`${API_BASE_URL}${path}`, {
      signal,
      credentials: 'include', // send cookies (sessions, Phase 1)
      headers: { Accept: 'application/json' },
    })
  } catch (err) {
    if (signal?.aborted) throw err // caller cancelled; not a real failure
    throw new ApiError(0, 'NETWORK_ERROR', 'Cannot reach the server')
  }

  let body: unknown
  try {
    body = await res.json()
  } catch {
    throw new ApiError(res.status, 'INVALID_RESPONSE', `Unexpected response (HTTP ${res.status})`)
  }

  if (!isEnvelope(body)) {
    throw new ApiError(res.status, 'INVALID_RESPONSE', `Unexpected response (HTTP ${res.status})`)
  }
  if (!body.success) {
    throw new ApiError(res.status, body.error.code, body.error.message, body.error.details)
  }
  return body.data as T
}
