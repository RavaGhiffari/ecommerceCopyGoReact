import type { ProductSort } from '../../types/marketplace'

export const SORT_OPTIONS: { value: ProductSort; label: string }[] = [
  { value: 'popular', label: 'Terlaris' },
  { value: 'newest', label: 'Terbaru' },
  { value: 'rating', label: 'Rating tertinggi' },
  { value: 'price_asc', label: 'Harga terendah' },
  { value: 'price_desc', label: 'Harga tertinggi' },
]

export const DEFAULT_SORT: ProductSort = 'popular'
export const PAGE_SIZE = 12

/** URL values are user input: anything unexpected falls back to a default. */
export function parseSort(value: string | null): ProductSort {
  return SORT_OPTIONS.some((o) => o.value === value) ? (value as ProductSort) : DEFAULT_SORT
}

export function parsePage(value: string | null): number {
  const n = Number.parseInt(value ?? '', 10)
  return Number.isFinite(n) && n >= 1 ? n : 1
}
