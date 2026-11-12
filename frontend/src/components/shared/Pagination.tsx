import { ChevronLeft, ChevronRight } from 'lucide-react'

interface PaginationProps {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
}

export function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  if (totalPages <= 1) return null

  const button =
    'inline-flex min-w-9 items-center justify-center rounded-md border px-2 py-1.5 text-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600'

  return (
    <nav aria-label="Halaman" className="mt-8 flex items-center justify-center gap-1">
      <button
        type="button"
        className={`${button} border-slate-300 bg-white text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40`}
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
        aria-label="Halaman sebelumnya"
      >
        <ChevronLeft className="size-4" aria-hidden="true" />
      </button>

      {Array.from({ length: totalPages }, (_, i) => i + 1).map((n) => (
        <button
          key={n}
          type="button"
          onClick={() => onPageChange(n)}
          aria-current={n === page ? 'page' : undefined}
          className={`${button} ${
            n === page
              ? 'border-emerald-600 bg-emerald-600 text-white'
              : 'border-slate-300 bg-white text-slate-700 hover:bg-slate-50'
          }`}
        >
          {n}
        </button>
      ))}

      <button
        type="button"
        className={`${button} border-slate-300 bg-white text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40`}
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
        aria-label="Halaman berikutnya"
      >
        <ChevronRight className="size-4" aria-hidden="true" />
      </button>
    </nav>
  )
}
