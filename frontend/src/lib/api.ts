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
  readonly details?: unknown // e.g. { email: "is already registered" } for 422/409

  constructor(status: number, code: string, message: string, details?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
  }

  /** Per-field messages from a 422 / 409 response, if present. */
  fieldErrors(): Record<string, string> {
    if (typeof this.details !== 'object' || this.details === null) return {}
    return Object.fromEntries(
      Object.entries(this.details).filter((entry): entry is [string, string] => typeof entry[1] === 'string'),
    )
  }
}

function isEnvelope(body: unknown): body is ApiEnvelope<unknown> {
  return typeof body === 'object' && body !== null && 'success' in body
}

type Method = 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'

/**
 * Calls the API and returns the envelope's `data`, or throws ApiError.
 * Returns undefined for 204 No Content.
 *
 * `credentials: 'include'` sends the HttpOnly session cookie. JavaScript can
 * never read that cookie; the browser attaches it by itself.
 */
export async function apiRequest<T>(
  method: Method,
  path: string,
  options: { body?: unknown; signal?: AbortSignal; headers?: Record<string, string> } = {},
): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json', ...options.headers }
  if (options.body !== undefined) headers['Content-Type'] = 'application/json'

  let res: Response
  try {
    res = await fetch(`${API_BASE_URL}${path}`, {
      method,
      signal: options.signal,
      credentials: 'include',
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
    })
  } catch (err) {
    if (options.signal?.aborted) throw err // caller cancelled; not a real failure
    throw new ApiError(0, 'NETWORK_ERROR', 'Tidak dapat terhubung ke server')
  }

  if (res.status === 204) return undefined as T

  let body: unknown
  try {
    body = await res.json()
  } catch {
    throw new ApiError(res.status, 'INVALID_RESPONSE', `Respons tidak terduga (HTTP ${res.status})`)
  }

  if (!isEnvelope(body)) {
    throw new ApiError(res.status, 'INVALID_RESPONSE', `Respons tidak terduga (HTTP ${res.status})`)
  }
  if (!body.success) {
    throw new ApiError(res.status, body.error.code, body.error.message, body.error.details)
  }
  return body.data as T
}

/** GET shortcut, kept for existing callers. */
export function apiGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  return apiRequest<T>('GET', path, { signal })
}
