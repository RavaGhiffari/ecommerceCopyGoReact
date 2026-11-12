import type { ReactNode } from 'react'
import { AlertTriangle, PackageSearch } from 'lucide-react'
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
    <div className="flex flex-col items-center rounded-xl border border-dashed border-slate-300 bg-white px-6 py-14 text-center">
      <PackageSearch className="size-10 text-slate-400" aria-hidden="true" />
      <h2 className="mt-3 text-lg font-semibold text-slate-900">{title}</h2>
      {description && <p className="mt-1 max-w-md text-sm text-slate-600">{description}</p>}
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
      className="flex flex-col items-center rounded-xl border border-red-200 bg-red-50 px-6 py-12 text-center"
    >
      <AlertTriangle className="size-10 text-red-500" aria-hidden="true" />
      <h2 className="mt-3 text-lg font-semibold text-red-900">{title}</h2>
      <p className="mt-1 max-w-md text-sm text-red-800">{error.message}</p>
      <p className="mt-1 font-mono text-xs text-red-700">{error.code}</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="mt-5 rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-600 focus-visible:ring-offset-2"
        >
          Coba lagi
        </button>
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
        <div key={i} className="animate-pulse overflow-hidden rounded-xl border border-slate-200 bg-white">
          <div className="aspect-square bg-slate-200" />
          <div className="space-y-2 p-3">
            <div className="h-3 w-5/6 rounded bg-slate-200" />
            <div className="h-3 w-2/3 rounded bg-slate-200" />
            <div className="h-4 w-1/2 rounded bg-slate-200" />
          </div>
        </div>
      ))}
    </div>
  )
}
