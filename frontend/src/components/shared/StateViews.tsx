import type { ReactNode } from 'react'
import { AlertTriangle, PackageSearch, RotateCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { ApiError } from '../../lib/api'

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string
  description?: string
  action?: ReactNode
}) {
  return (
    <div className="flex flex-col items-center rounded-xl border border-dashed bg-card px-6 py-14 text-center">
      <PackageSearch className="size-10 text-muted-foreground" aria-hidden="true" />
      <h2 className="mt-3 text-lg font-semibold">{title}</h2>
      {description && <p className="mt-1 max-w-md text-sm text-muted-foreground">{description}</p>}
      {action && <div className="mt-5">{action}</div>}
    </div>
  )
}

export function ErrorState({
  error,
  onRetry,
  title = 'Gagal memuat data',
}: {
  error: ApiError
  onRetry?: () => void
  title?: string
}) {
  return (
    <div
      role="alert"
      className="flex flex-col items-center rounded-xl border border-destructive/30 bg-destructive/5 px-6 py-12 text-center"
    >
      <AlertTriangle className="size-10 text-destructive" aria-hidden="true" />
      <h2 className="mt-3 text-lg font-semibold text-destructive">{title}</h2>
      <p className="mt-1 max-w-md text-sm text-foreground/80">{error.message}</p>
      <p className="mt-1 font-mono text-xs text-muted-foreground">{error.code}</p>
      {onRetry && (
        <Button variant="destructive" size="lg" className="mt-5" onClick={onRetry}>
          <RotateCw aria-hidden="true" /> Coba lagi
        </Button>
      )}
    </div>
  )
}

export function ProductGridSkeleton({ count = 8 }: { count?: number }) {
  return (
    <div
      className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-4"
      role="status"
      aria-label="Memuat produk"
    >
      {Array.from({ length: count }, (_, i) => (
        <Card key={i} className="gap-0 py-0">
          <Skeleton className="aspect-square rounded-none" />
          <div className="space-y-2 p-3">
            <Skeleton className="h-3 w-5/6" />
            <Skeleton className="h-3 w-2/3" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        </Card>
      ))}
    </div>
  )
}
