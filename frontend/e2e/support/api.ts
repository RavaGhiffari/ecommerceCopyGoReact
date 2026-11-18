// Calls the API with a page's cookies, for setup and for checks the UI does
// not show. Requests go through Vite like the browser's, with the same Origin
// header, so the CSRF check sees an ordinary same-origin request.
import { expect, type Page } from '@playwright/test'
import { BASE_URL } from '../../playwright.config'
import { SURABAYA_ADDRESS } from './data'

type Method = 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'

export interface ApiResult {
  status: number
  // The envelope: { success, data, message } or { success: false, error }.
  // Typed loosely on purpose: specs read only the fields they check, and the
  // unit-tested TS types in src/types describe the full shapes.
  body: any
}

export class Api {
  private readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  async call(method: Method, path: string, data?: unknown, headers: Record<string, string> = {}): Promise<ApiResult> {
    const res = await this.page.request.fetch(`${BASE_URL}/api/v1${path}`, {
      method,
      data,
      headers: { Origin: BASE_URL, ...headers },
    })
    return { status: res.status(), body: res.status() === 204 ? null : await res.json() }
  }

  /** The envelope's data; fails the test on any non-2xx answer. */
  async data(method: Method, path: string, payload?: unknown, headers?: Record<string, string>): Promise<any> {
    const res = await this.call(method, path, payload, headers)
    expect(res.status, `${method} ${path} -> ${JSON.stringify(res.body)}`).toBeLessThan(300)
    return res.body?.data
  }

  get = (path: string) => this.data('GET', path)

  /** Signs up a new customer; the page's browser context is then logged in. */
  async register(name: string, email: string, password: string) {
    return this.data('POST', '/auth/register', { name, email, password })
  }

  async login(email: string, password: string) {
    return this.data('POST', '/auth/login', { email, password })
  }

  async clearCart() {
    const cart = await this.get('/cart')
    for (const group of cart.stores) for (const item of group.items) await this.data('DELETE', `/cart/items/${item.id}`)
  }

  async addAddress(overrides: Partial<typeof SURABAYA_ADDRESS> = {}) {
    return this.data('POST', '/me/addresses', { ...SURABAYA_ADDRESS, ...overrides })
  }

  /**
   * Buys one unit of the product at `productPath` and pays for it. Returns
   * the checkout (with its orders). Everything goes through the real API:
   * cart -> checkout (Idempotency-Key) -> simulated payment.
   */
  async buyAndPay(productPath: string, addressId: string) {
    const product = await this.get(productPath)
    await this.clearCart()
    await this.data('POST', '/cart/items', { variantId: product.variants[0].id, quantity: 1 })
    const checkout = await this.data(
      'POST',
      '/checkout',
      { addressId, shipping: { [product.store.id]: 'regular' } },
      { 'Idempotency-Key': crypto.randomUUID() },
    )
    return this.data('POST', `/payments/${checkout.payment.id}/simulate`, {
      outcome: 'success',
      eventId: crypto.randomUUID(),
    })
  }

  /** Seller side: moves an order along paid -> processing -> ... -> `to`. */
  async advanceOrder(orderId: string, to: 'processing' | 'shipped' | 'delivered') {
    const steps = ['processing', 'shipped', 'delivered'] as const
    let order
    for (const status of steps.slice(0, steps.indexOf(to) + 1)) {
      order = await this.data('PATCH', `/seller/orders/${orderId}/status`, { status })
    }
    return order
  }
}

export const apiOf = (page: Page) => new Api(page)
