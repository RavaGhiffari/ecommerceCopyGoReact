// Demo data created by `seed-dev` (backend/cmd/seed-dev). Local accounts only.

export const SELLER_PASSWORD = 'penjual-123'

/** Gadget Nusantara, Jakarta Barat. */
export const SELLER_GADGET = 'penjual1@contoh.id'
/** Batik Kita, Surabaya (the only demo store that offers Same Day to Surabaya). */
export const SELLER_BATIK = 'penjual2@contoh.id'
/** Sportivo Store; the admin spec suspends and reactivates this account. */
export const SELLER_SPORTIVO = 'penjual4@contoh.id'

export const ADMIN_EMAIL = 'e2e-admin@contoh.id'

export const CABLE = '/stores/gadget-nusantara/products/kabel-usb-c-braided'
export const CABLE_NAME = 'Kabel USB-C Braided'
export const TSHIRT = '/stores/batik-kita/products/kaos-polos-cotton-combed-24s'
export const TSHIRT_NAME = 'Kaos Polos Cotton Combed 24s'

/** A fresh address in Surabaya (API field names). */
export const SURABAYA_ADDRESS = {
  label: 'Rumah',
  recipientName: 'Pembeli E2E',
  phone: '081234567890',
  addressLine: 'Jl. Tunjungan No. 1',
  city: 'Surabaya',
  province: 'Jawa Timur',
  postalCode: '60261',
}

/** Same formatting as the app (Intl, non-breaking space after "Rp"). */
export const rupiah = (n: number | string) =>
  new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(n))

/** A new address for every call, so reruns without a reset never collide. */
export function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 7)}@contoh.id`
}
