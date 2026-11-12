const rupiah = new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
})

export function formatRupiah(amount: number): string {
  return rupiah.format(amount)
}

/** "Rp49.000" or "Rp49.000 - Rp59.000" when variants differ in price. */
export function formatPriceRange(lowest: number, highest: number): string {
  return lowest === highest
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
