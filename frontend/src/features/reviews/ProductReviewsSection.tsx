import { useState } from 'react'
import { MessageSquare } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Pagination } from '../../components/shared/Pagination'
import { ErrorState } from '../../components/shared/StateViews'
import { Stars } from '../../components/shared/Stars'
import { useAsync } from '../../hooks/useAsync'
import { formatDateTime } from '../../lib/format'
import { listProductReviews } from '../../services/reviews'
import type { PublicReview, ReviewSummary } from '../../types/reviews'

const PAGE_SIZE = 5

/**
 * "Ulasan pembeli" on the product page: summary, distribution, list.
 * Mount it with key={productId} so another product starts from page 1.
 */
export function ProductReviewsSection({ productId }: { productId: string }) {
  const [page, setPage] = useState(1)
  const { state, reload, lastData } = useAsync(
    () => listProductReviews(productId, page, PAGE_SIZE),
    `reviews:${productId}:${page}`,
  )
  // While the next page loads, keep showing the previous one.
  const data = state.status === 'success' ? state.data : lastData

  let body
  if (state.status === 'error') {
    body = <ErrorState error={state.error} onRetry={reload} title="Gagal memuat ulasan" />
  } else if (!data) {
    body = (
      <div className="grid gap-4 md:grid-cols-[16rem_1fr]" role="status" aria-label="Memuat ulasan">
        <Skeleton className="h-40 rounded-xl" />
        <Skeleton className="h-40 rounded-xl" />
      </div>
    )
  } else if (data.summary.count === 0) {
    body = (
      <Card className="flex flex-col items-center gap-2 px-6 py-10 text-center">
        <MessageSquare className="size-8 text-muted-foreground" aria-hidden="true" />
        <p className="font-medium">Belum ada ulasan</p>
        <p className="max-w-sm text-sm text-muted-foreground">
          Ulasan muncul setelah pembeli menerima pesanannya dan memberi rating.
        </p>
      </Card>
    )
  } else {
    body = (
      <div className="grid gap-4 md:grid-cols-[16rem_1fr] md:items-start">
        <SummaryCard summary={data.summary} />
        <div aria-busy={state.status === 'loading'}>
          <ul className="divide-y rounded-xl bg-card ring-1 ring-foreground/10">
            {data.items.map((r) => (
              <ReviewRow key={r.id} review={r} />
            ))}
          </ul>
          <Pagination page={data.page} totalPages={data.totalPages} onPageChange={setPage} />
        </div>
      </div>
    )
  }

  return (
    <section aria-labelledby="product-reviews" id="ulasan" className="scroll-mt-24">
      <h2 id="product-reviews" className="mb-3 text-xl font-semibold">
        Ulasan pembeli
      </h2>
      {body}
    </section>
  )
}

function SummaryCard({ summary }: { summary: ReviewSummary }) {
  return (
    <Card className="block p-5">
      <p className="flex items-baseline gap-1">
        <span className="text-4xl font-bold">{summary.average.toFixed(1).replace('.', ',')}</span>
        <span className="text-muted-foreground">/5</span>
      </p>
      <Stars value={summary.average} className="mt-1" />
      <p className="mt-1 text-sm text-muted-foreground">{summary.count} ulasan</p>
      <ul className="mt-4 space-y-1.5" aria-label="Sebaran rating">
        {([5, 4, 3, 2, 1] as const).map((star) => {
          const n = summary.distribution[String(star) as keyof ReviewSummary['distribution']]
          const pct = summary.count ? Math.round((n / summary.count) * 100) : 0
          return (
            <li key={star} className="flex items-center gap-2 text-xs">
              <span className="w-3 text-right text-muted-foreground">{star}</span>
              <span className="h-2 flex-1 overflow-hidden rounded-full bg-muted" aria-hidden="true">
                <span className="block h-full rounded-full bg-amber-400" style={{ width: `${pct}%` }} />
              </span>
              <span className="w-6 text-right text-muted-foreground">{n}</span>
              <span className="sr-only">
                {star} bintang: {n} ulasan
              </span>
            </li>
          )
        })}
      </ul>
    </Card>
  )
}

function ReviewRow({ review: r }: { review: PublicReview }) {
  return (
    <li className="space-y-1.5 p-4">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <Stars value={r.rating} />
        <span className="text-sm font-medium">{r.reviewerName}</span>
        <span className="text-xs text-muted-foreground">
          {formatDateTime(r.createdAt)}
          {r.edited && ' · diubah'}
        </span>
      </div>
      <p className="text-xs text-muted-foreground">Varian: {r.variantName}</p>
      {r.comment ? (
        // Plain text: React escapes it; newlines are kept by whitespace-pre-line.
        <p className="text-sm whitespace-pre-line">{r.comment}</p>
      ) : (
        <p className="text-sm text-muted-foreground italic">Tanpa komentar.</p>
      )}
    </li>
  )
}
