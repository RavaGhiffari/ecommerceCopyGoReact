const rupiah = new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
})

/**
 * Formats rupiah for DISPLAY. Accepts the API's decimal strings ("349000.00").
 * Converting to a JS number is fine for showing a price; never add up money
 * client-side for anything the server will charge (the server recomputes).
 */
export function formatRupiah(amount: number | string): string {
  return rupiah.format(typeof amount === 'string' ? Number(amount) : amount)
}

/** "Rp49.000" or "Rp49.000 - Rp59.000" when variants differ in price. */
export function formatPriceRange(lowest: number | string, highest: number | string): string {
  return Number(lowest) === Number(highest)
    ? formatRupiah(lowest)
    : `${formatRupiah(lowest)} - ${formatRupiah(highest)}`
}

/** 950 -> "950", 1250 -> "1,2 rb", 21000 -> "21 rb", 1500000 -> "1,5 jt". */
export function formatCompactCount(n: number): string {
  const trim = (value: number) => value.toFixed(1).replace('.', ',').replace(/,0$/, '')
  if (n >= 1_000_000) return `${trim(n / 1_000_000)} jt`
  if (n >= 1_000) return `${trim(n / 1_000)} rb`
  return String(n)
}

export function formatMonthYear(iso: string): string {
  return new Intl.DateTimeFormat('id-ID', { month: 'long', year: 'numeric' }).format(new Date(iso))
}

const dateTime = new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' })

/** "15 Nov 2026, 14.05" in the viewer's time zone. */
export function formatDateTime(iso: string): string {
  return dateTime.format(new Date(iso))
}

/** Milliseconds -> "29:58" or "1:02:03". Negative values show "0:00". */
export function formatCountdown(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000))
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = String(total % 60).padStart(2, '0')
  return h > 0 ? `${h}:${String(m).padStart(2, '0')}:${s}` : `${m}:${s}`
}
