import { useSearchParams } from 'react-router-dom'
import type { ProductSort } from '../../types/marketplace'
import { DEFAULT_SORT, parsePage, parseSort } from './query'

/**
 * Search / sort / page live in the URL (?q=...&sort=...&page=...) so a result
 * list can be bookmarked, shared, and survives the back button.
 */
export function useListParams() {
  const [params, setParams] = useSearchParams()

  const q = (params.get('q') ?? '').trim()
  const sort = parseSort(params.get('sort'))
  const page = parsePage(params.get('page'))

  // Copies the existing params first so unrelated ones stay.
  const update = (changes: Record<string, string | null>) => {
    const next = new URLSearchParams(params)
    for (const [key, value] of Object.entries(changes)) {
      if (value === null || value === '') next.delete(key)
      else next.set(key, value)
    }
    setParams(next)
  }

  return {
    q,
    sort,
    page,
    // Changing the sort order starts again from the first page.
    setSort: (s: ProductSort) => update({ sort: s === DEFAULT_SORT ? null : s, page: null }),
    setPage: (p: number) => update({ page: p === 1 ? null : String(p) }),
  }
}
