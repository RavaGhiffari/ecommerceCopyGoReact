import { Skeleton } from '@/components/ui/skeleton'
import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { ErrorState } from '../../components/shared/StateViews'
import { ApiError } from '../../lib/api'
import { useAuth } from './authContext'

/**
 * Sends anonymous visitors to /login and back afterwards.
 *
 * UX only: it saves a round trip to a page that would show nothing. The data
 * on that page is protected by the backend (RequireAuth middleware), which
 * answers 401 no matter what the browser does.
 */
export function RequireAuth({ children }: { children: ReactNode }) {
  const { status, refresh } = useAuth()
  const location = useLocation()

  if (status === 'loading') {
    return <Skeleton className="h-40 rounded-xl" role="status" aria-label="Memeriksa sesi" />
  }
  if (status === 'error') {
    return (
      <ErrorState
        error={new ApiError(0, 'NETWORK_ERROR', 'Tidak dapat memeriksa sesi login. Pastikan backend berjalan.')}
        onRetry={refresh}
      />
    )
  }
  if (status === 'anonymous') {
    const next = encodeURIComponent(location.pathname + location.search)
    return <Navigate to={`/login?next=${next}`} replace />
  }
  return children
}
