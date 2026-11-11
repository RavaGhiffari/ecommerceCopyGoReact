import { useCallback, useEffect, useState } from 'react'
import { apiGet, ApiError } from '../../lib/api'

export type CheckState<T> =
  | { kind: 'loading' }
  | { kind: 'ok'; data: T; ms: number }
  | { kind: 'error'; error: ApiError; ms: number }

/**
 * Calls one GET endpoint on mount and whenever `reload` is called.
 *
 * Deliberately plain useState/useEffect: TanStack Query arrives with the real
 * data screens (Phase 3); adding it for one status page would be premature.
 */
export function useApiCheck<T>(path: string) {
  const [state, setState] = useState<CheckState<T>>({ kind: 'loading' })
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    // AbortController cancels the request if the component unmounts (or React
    // StrictMode runs the effect twice in dev), so a stale response can't
    // overwrite newer state.
    const controller = new AbortController()
    const started = performance.now()

    apiGet<T>(path, controller.signal)
      .then((data) => {
        setState({ kind: 'ok', data, ms: Math.round(performance.now() - started) })
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        const error =
          err instanceof ApiError ? err : new ApiError(0, 'UNKNOWN', 'Unexpected error')
        setState({ kind: 'error', error, ms: Math.round(performance.now() - started) })
      })

    return () => controller.abort()
  }, [path, attempt])

  const reload = useCallback(() => {
    setState({ kind: 'loading' })
    setAttempt((n) => n + 1)
  }, [])

  return { state, reload }
}
