import type { CheckoutStatus, PaymentStatus } from '../../types/checkout'
import type { OrderStatus, ShippingMethod } from '../../types/orders'

export const ORDER_STATUS_LABEL: Record<OrderStatus, string> = {
  pending_payment: 'Menunggu pembayaran',
  paid: 'Dibayar',
  processing: 'Diproses',
  shipped: 'Dikirim',
  delivered: 'Selesai',
  cancelled: 'Dibatalkan',
  refunded: 'Dana dikembalikan',
}

/** Badge colours per status: waiting = amber, in progress = blue, done = green, stopped = grey/red. */
export const ORDER_STATUS_TONE: Record<OrderStatus, string> = {
  pending_payment: 'bg-amber-100 text-amber-900',
  paid: 'bg-sky-100 text-sky-900',
  processing: 'bg-sky-100 text-sky-900',
  shipped: 'bg-indigo-100 text-indigo-900',
  delivered: 'bg-emerald-100 text-emerald-900',
  cancelled: 'bg-muted text-muted-foreground',
  refunded: 'bg-rose-100 text-rose-900',
}

export const ORDER_STATUSES = Object.keys(ORDER_STATUS_LABEL) as OrderStatus[]

export const CHECKOUT_STATUS_LABEL: Record<CheckoutStatus, string> = {
  pending_payment: 'Menunggu pembayaran',
  paid: 'Lunas',
  cancelled: 'Dibatalkan',
  expired: 'Waktu pembayaran habis',
}

export const PAYMENT_STATUS_LABEL: Record<PaymentStatus, string> = {
  pending: 'Menunggu',
  succeeded: 'Berhasil',
  failed: 'Gagal',
  cancelled: 'Dibatalkan',
  expired: 'Kedaluwarsa',
}

export const SHIPPING_LABEL: Record<ShippingMethod, string> = {
  regular: 'Reguler',
  express: 'Express',
  same_day: 'Same Day',
}

export const SHIPPING_HINT: Record<ShippingMethod, string> = {
  regular: 'Estimasi 3–5 hari',
  express: 'Estimasi 1–2 hari',
  same_day: 'Tiba hari ini (satu kota)',
}

/**
 * The seller's next step for each status. A UI copy of the seller rows in
 * backend/internal/modules/orders/transitions.go: it only decides which
 * button to show. The server checks the transition table again.
 */
export const SELLER_NEXT: Partial<Record<OrderStatus, { to: OrderStatus; action: string; noteLabel: string }>> = {
  paid: { to: 'processing', action: 'Proses pesanan', noteLabel: 'Catatan (opsional)' },
  processing: { to: 'shipped', action: 'Tandai dikirim', noteLabel: 'Nomor resi / catatan (opsional)' },
  shipped: { to: 'delivered', action: 'Tandai selesai', noteLabel: 'Catatan (opsional)' },
}

/** The buyer may cancel (simulated refund) only while the seller has not started. */
export const buyerCanRefund = (status: OrderStatus) => status === 'paid'

// Default history notes written by the backend (orders/service_seller.go,
// orders/repository.go, checkout/lifecycle.go, orders/service.go). Shown in
// Indonesian; notes a seller typed are shown as they are.
const DEFAULT_NOTES: Record<string, string> = {
  'Order created': 'Pesanan dibuat',
  'Payment received': 'Pembayaran diterima',
  'Payment failed': 'Pembayaran gagal',
  'Payment time ran out': 'Waktu pembayaran habis',
  'Cancelled by the buyer': 'Dibatalkan pembeli',
  'Cancelled by the buyer; refund simulated': 'Dibatalkan pembeli; dana dikembalikan (simulasi)',
  'The seller is preparing the order': 'Penjual sedang menyiapkan pesanan',
  'Order shipped': 'Pesanan dikirim',
  'Order delivered': 'Pesanan diterima',
}

export const historyNote = (note: string) => DEFAULT_NOTES[note] ?? note
