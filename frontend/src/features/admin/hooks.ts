import { useCallback, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useAuth } from '../auth/authContext'

/** True when the signed-in user has the admin role (header links only). */
export function useIsAdmin() {
  const { user } = useAuth()
  return user?.roles.includes('admin') ?? false
}

/**
 * Search, status filter and page of an admin list, kept in the URL
 * (?q=&status=&page=) so a filtered list survives reload and the back button.
 * An unknown status in the URL is ignored rather than sent to the API.
 */
export function useAdminListParams<S extends string>(statuses: readonly S[]) {
  const [params, setParams] = useSearchParams()
  const q = (params.get('q') ?? '').trim()
  const raw = params.get('status') ?? ''
  const status = (statuses as readonly string[]).includes(raw) ? (raw as S) : undefined
  const page = Math.max(1, Number.parseInt(params.get('page') ?? '', 10) || 1)

  // A new search or filter starts again from page 1.
  const update = (next: { q?: string; status?: S | ''; page?: number }) => {
    const out = new URLSearchParams()
    const nq = next.q ?? q
    const ns = next.status ?? status ?? ''
    const np = next.page ?? 1
    if (nq) out.set('q', nq)
    if (ns) out.set('status', ns)
    if (np > 1) out.set('page', String(np))
    setParams(out)
  }

  return { q, status, page, update }
}

/**
 * Rows the page changed itself (e.g. a suspended user returned by PATCH),
 * shown over the loaded list without refetching it. The patches belong to one
 * list `key` and are dropped when the key (filter, page) changes.
 */
export function usePatchedRows<T extends { id: string }>(key: string) {
  const [patches, setPatches] = useState<{ key: string; rows: Record<string, T> }>({ key, rows: {} })
  const rows = patches.key === key ? patches.rows : {}

  const apply = (list: T[]) => list.map((row) => rows[row.id] ?? row)
  const patch = useCallback(
    (row: T) => setPatches((p) => ({ key, rows: { ...(p.key === key ? p.rows : {}), [row.id]: row } })),
    [key],
  )
  return { apply, patch }
}
