import { describe, expect, it } from 'vitest'
import type { OrderStatus } from '../../types/orders'
import { ORDER_STATUSES, ORDER_STATUS_LABEL, ORDER_STATUS_TONE, SELLER_NEXT, buyerCanRefund, historyNote } from './status'

// A UI copy of the transition table in backend/internal/modules/orders/transitions.go.
// It only decides which buttons appear; these tests pin it to the backend rules.
describe('order status UI rules', () => {
  it('knows all 7 backend statuses, each with a label and a colour', () => {
    expect([...ORDER_STATUSES].sort()).toEqual(
      ['cancelled', 'delivered', 'paid', 'pending_payment', 'processing', 'refunded', 'shipped'].sort(),
    )
    for (const s of ORDER_STATUSES) {
      expect(ORDER_STATUS_LABEL[s]).toBeTruthy()
      expect(ORDER_STATUS_TONE[s]).toBeTruthy()
    }
  })

  it('walks the seller through paid -> processing -> shipped -> delivered and no further', () => {
    const path: OrderStatus[] = ['paid']
    for (let s = SELLER_NEXT.paid; s; s = SELLER_NEXT[s.to]) path.push(s.to)
    expect(path).toEqual(['paid', 'processing', 'shipped', 'delivered'])

    for (const s of ['pending_payment', 'delivered', 'cancelled', 'refunded'] as OrderStatus[]) {
      expect(SELLER_NEXT[s], `seller action offered for ${s}`).toBeUndefined()
    }
  })

  it('lets the buyer cancel (refund) only a paid order', () => {
    expect(ORDER_STATUSES.filter(buyerCanRefund)).toEqual(['paid'])
  })

  it('translates the default history notes and keeps seller-typed notes', () => {
    expect(historyNote('Order shipped')).toBe('Pesanan dikirim')
    expect(historyNote('Resi JNE 123')).toBe('Resi JNE 123')
  })
})
