import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../lib/api'

export type AsyncState<T> =
  | { status: 'loading' }
  | { status: 'success'; data: T }
  | { status: 'error'; error: ApiError }

type Settled<T> = Exclude<AsyncState<T>, { status: 'loading' }>

/**
 * Runs `load` and reports loading / success / error.
 *
 * `key` identifies the request: when it changes, the previous result is
 * discarded (state becomes "loading") and `load` runs again. Build it from
 * every input `load` depends on, e.g. `products:${JSON.stringify(query)}`.
 *
 * Plain React on purpose. TanStack Query arrives together with the real API
 * (it adds caching and retries that mock data does not need).
 */
export function useAsync<T>(load: () => Promise<T>, key: string) {
  const [attempt, setAttempt] = useState(0)
  const requestKey = `${key}#${attempt}`

  // The result remembers which request produced it. A result for an older key
  // is ignored, so there is no synchronous setState inside the effect.
  const [result, setResult] = useState<{ key: string; state: Settled<T> } | null>(null)
  // The latest successful data of ANY key, so a view can keep showing the
  // previous result while the next one loads (e.g. a re-calculated quote).
  const [lastData, setLastData] = useState<T | undefined>(undefined)

  // Always call the latest `load` without making it an effect dependency.
  const loadRef = useRef(load)
  useEffect(() => {
    loadRef.current = load
  })

  useEffect(() => {
    let cancelled = false
    loadRef
      .current()
      .then((data) => {
        if (cancelled) return
        setResult({ key: requestKey, state: { status: 'success', data } })
        setLastData(data)
      })
      .catch((err: unknown) => {
        if (cancelled) return
        const error =
          err instanceof ApiError ? err : new ApiError(0, 'UNKNOWN', 'Terjadi kesalahan tak terduga.')
        setResult({ key: requestKey, state: { status: 'error', error } })
      })
    return () => {
      cancelled = true // a newer request, or unmount, supersedes this one
    }
  }, [requestKey])

  const reload = useCallback(() => setAttempt((n) => n + 1), [])

  const state: AsyncState<T> =
    result !== null && result.key === requestKey ? result.state : { status: 'loading' }

  return { state, reload, lastData }
}

/**
 * useAsync for a single resource that the page can also change, e.g. an order
 * whose cancel endpoint returns the updated order. `replace` shows the
 * server's answer without a reload; `reload` drops it and fetches again.
 */
export function useAsyncResource<T>(load: () => Promise<T>, key: string) {
  const { state, reload: reloadLoaded } = useAsync(load, key)
  const [replaced, setReplaced] = useState<{ key: string; data: T } | null>(null)

  const current: AsyncState<T> =
    replaced !== null && replaced.key === key ? { status: 'success', data: replaced.data } : state
  const replace = useCallback((data: T) => setReplaced({ key, data }), [key])
  const reload = useCallback(() => {
    setReplaced(null)
    reloadLoaded()
  }, [reloadLoaded])

  return { state: current, replace, reload }
}
