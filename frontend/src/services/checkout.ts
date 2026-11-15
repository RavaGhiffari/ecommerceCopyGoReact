// Checkout and payment simulation API. Everything is scoped to the session
// user on the server; another user's checkout id is a 404.
import { apiRequest } from '../lib/api'
import type { Checkout, CheckoutInput, PaymentOutcome, Quote } from '../types/checkout'

const enc = encodeURIComponent

/** Read-only preview: prices, fees and totals computed by the server. */
export const quoteCheckout = (input: CheckoutInput, signal?: AbortSignal) =>
  apiRequest<Quote>('POST', '/api/v1/checkout/quote', { body: input, signal })

/**
 * Places the order. `idempotencyKey` must stay the same for retries of the
 * same attempt (double click, timeout), so the server returns the first
 * checkout instead of creating a second one.
 */
export const placeCheckout = (input: CheckoutInput, idempotencyKey: string) =>
  apiRequest<Checkout>('POST', '/api/v1/checkout', {
    body: input,
    headers: { 'Idempotency-Key': idempotencyKey },
  })

export const getCheckout = (id: string, signal?: AbortSignal) =>
  apiRequest<Checkout>('GET', `/api/v1/checkout/${enc(id)}`, { signal })

/** Cancels the whole checkout while it is unpaid; reserved stock is released. */
export const cancelCheckout = (id: string) => apiRequest<Checkout>('POST', `/api/v1/checkout/${enc(id)}/cancel`)

/**
 * Plays the payment gateway's notification. `eventId` is the gateway's
 * notification id: sending the same one again is a harmless repeat.
 */
export const simulatePayment = (paymentId: string, outcome: PaymentOutcome, eventId: string) =>
  apiRequest<Checkout>('POST', `/api/v1/payments/${enc(paymentId)}/simulate`, { body: { outcome, eventId } })
