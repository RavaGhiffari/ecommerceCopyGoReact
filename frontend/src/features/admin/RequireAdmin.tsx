import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { RequireAuth } from '../auth/RequireAuth'
import { useAuth } from '../auth/authContext'

/**
 * UX only: anonymous visitors go to /login, signed-in non-admins see a short
 * notice. Every /api/v1/admin route checks the admin role itself (403), so
 * hiding these pages protects nothing; it only avoids a page of errors.
 */
export function RequireAdmin({ children }: { children: ReactNode }) {
  return (
    <RequireAuth>
      <AdminOnly>{children}</AdminOnly>
    </RequireAuth>
  )
}

function AdminOnly({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  if (user?.roles.includes('admin')) return children
  return (
    <Card className="mx-auto max-w-md items-center gap-3 p-8 text-center">
      <ShieldAlert className="size-10 text-muted-foreground" aria-hidden="true" />
      <h1 className="text-lg font-semibold">Halaman khusus admin</h1>
      <p className="text-sm text-muted-foreground">Akunmu tidak punya akses ke panel admin.</p>
      <Button asChild variant="outline">
        <Link to="/">Kembali ke beranda</Link>
      </Button>
    </Card>
  )
}
