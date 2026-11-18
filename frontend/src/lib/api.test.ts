import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiRequest } from './api'

function respond(status: number, body?: unknown) {
  const fetchMock = vi.fn(async () =>
    body === undefined
      ? new Response(null, { status })
      : new Response(typeof body === 'string' ? body : JSON.stringify(body), {
          status,
          headers: { 'Content-Type': 'application/json' },
        }),
  )
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

async function caught(p: Promise<unknown>): Promise<ApiError> {
  try {
    await p
  } catch (err) {
    expect(err).toBeInstanceOf(ApiError)
    return err as ApiError
  }
  throw new Error('expected the request to fail')
}

afterEach(() => vi.unstubAllGlobals())

describe('apiRequest', () => {
  it('returns the envelope data and sends the session cookie', async () => {
    const fetchMock = respond(200, { success: true, data: { id: 'x' }, message: '' })

    await expect(apiRequest('GET', '/api/v1/me')).resolves.toEqual({ id: 'x' })

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/api/v1/me')
    expect(init.credentials).toBe('include')
    expect(init.body).toBeUndefined()
    expect((init.headers as Record<string, string>)['Content-Type']).toBeUndefined()
  })

  it('sends a JSON body with its content type and extra headers', async () => {
    const fetchMock = respond(201, { success: true, data: null, message: '' })

    await apiRequest('POST', '/api/v1/checkout', { body: { a: 1 }, headers: { 'Idempotency-Key': 'k' } })

    const init = (fetchMock.mock.calls[0] as unknown as [string, RequestInit])[1]
    expect(init.body).toBe('{"a":1}')
    expect(init.headers).toMatchObject({ 'Content-Type': 'application/json', 'Idempotency-Key': 'k' })
  })

  it('returns undefined for 204 No Content', async () => {
    respond(204)
    await expect(apiRequest('DELETE', '/api/v1/me/addresses/1')).resolves.toBeUndefined()
  })

  it('turns the error envelope into an ApiError with code and details', async () => {
    respond(422, {
      success: false,
      error: { code: 'VALIDATION_ERROR', message: 'invalid', details: { email: 'is required', extra: { nested: 1 } } },
    })

    const err = await caught(apiRequest('POST', '/api/v1/auth/register', { body: {} }))
    expect(err.status).toBe(422)
    expect(err.code).toBe('VALIDATION_ERROR')
    expect(err.fieldErrors()).toEqual({ email: 'is required' }) // non-string details are dropped
  })

  it('reports an unreachable server as NETWORK_ERROR with status 0', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => Promise.reject(new TypeError('Failed to fetch'))))

    const err = await caught(apiRequest('GET', '/api/v1/me'))
    expect(err.status).toBe(0)
    expect(err.code).toBe('NETWORK_ERROR')
  })

  it('rethrows the abort itself instead of reporting a network error', async () => {
    const controller = new AbortController()
    controller.abort()
    const abort = new DOMException('aborted', 'AbortError')
    vi.stubGlobal('fetch', vi.fn(async () => Promise.reject(abort)))

    await expect(apiRequest('GET', '/api/v1/me', { signal: controller.signal })).rejects.toBe(abort)
  })

  it.each([
    ['not JSON (e.g. a proxy error page)', '<html>Bad Gateway</html>'],
    ['JSON without the envelope', { hello: 'world' }],
  ])('treats %s as INVALID_RESPONSE', async (_name, body) => {
    respond(502, body)

    const err = await caught(apiRequest('GET', '/api/v1/me'))
    expect(err.code).toBe('INVALID_RESPONSE')
    expect(err.status).toBe(502)
  })
})
