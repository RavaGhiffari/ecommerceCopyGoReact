import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { RequireAuth } from '../auth/RequireAuth'
import { useAuth } from '../auth/authContext'

/**
 * UX only: sends signed-in non-sellers to /seller (where they can open a
 * store). The seller API checks the role itself on every request.
 */
export function RequireSeller({ children }: { children: ReactNode }) {
  return (
    <RequireAuth>
      <SellerOnly>{children}</SellerOnly>
    </RequireAuth>
  )
}

function SellerOnly({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  if (!user?.roles.includes('seller')) return <Navigate to="/seller" replace />
  return children
}
