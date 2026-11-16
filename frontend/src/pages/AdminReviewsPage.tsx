import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { Pagination } from '../components/shared/Pagination'
import { ItemRowSkeleton, LoadingRegion } from '../components/shared/Skeletons'
import { Stars } from '../components/shared/Stars'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { ActionButton, FilterChips, ResultCount } from '../features/admin/AdminParts'
import { useAdminListParams, usePatchedRows } from '../features/admin/hooks'
import { REVIEW_STATUSES, REVIEW_STATUS_LABEL } from '../features/admin/labels'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatDateTime } from '../lib/format'
import { cn } from '@/lib/utils'
import { listAdminReviews, setReviewStatus } from '../services/admin'
import type { AdminReview } from '../types/admin'

/**
 * /admin/reviews: hide or show reviews. A hidden review leaves the product
 * page and its rating average (the server adjusts the aggregate in the same
 * transaction); its author still sees it, marked as hidden.
 */
export function AdminReviewsPage() {
  usePageTitle('Ulasan · Admin')
  const { status, page, update } = useAdminListParams(REVIEW_STATUSES)
  const key = `admin-reviews:${status ?? ''}:${page}`
  const { state, reload } = useAsync(() => listAdminReviews(status, page), key)
  const { apply, patch } = usePatchedRows<AdminReview>(key)

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Moderasi ulasan</h1>
      <FilterChips
        label="Filter status"
        value={status}
        options={REVIEW_STATUSES.map((s) => ({ value: s, label: REVIEW_STATUS_LABEL[s] }))}
        onChange={(s) => update({ status: s })}
      />

      {state.status === 'loading' && (
        <LoadingRegion label="Memuat ulasan" className="block">
          <Card className="gap-0 py-0">
            {Array.from({ length: 4 }, (_, i) => (
              <ItemRowSkeleton key={i} thumb="hidden" />
            ))}
          </Card>
        </LoadingRegion>
      )}
      {state.status === 'error' && <ErrorState error={state.error} onRetry={reload} title="Gagal memuat ulasan" />}
      {state.status === 'success' && state.data.items.length === 0 && (
        <EmptyState title={status ? `Tidak ada ulasan berstatus “${REVIEW_STATUS_LABEL[status]}”` : 'Belum ada ulasan'} />
      )}
      {state.status === 'success' && state.data.items.length > 0 && (
        <>
          <ResultCount total={state.data.total} noun="ulasan" />
          <Card className="gap-0 py-0">
            <ul className="divide-y">
              {apply(state.data.items).map((r) => (
                <li key={r.id}>
                  <ReviewRow review={r} onChange={patch} />
                </li>
              ))}
            </ul>
          </Card>
          <Pagination
            page={state.data.page}
            totalPages={state.data.totalPages}
            onPageChange={(p) => update({ page: p })}
          />
        </>
      )}
    </div>
  )
}

function ReviewRow({ review: r, onChange }: { review: AdminReview; onChange: (r: AdminReview) => void }) {
  const hidden = r.status === 'hidden'
  return (
    <article className={cn('flex flex-col gap-3 p-4 sm:flex-row sm:items-start', hidden && 'bg-muted/40')}>
      <div className="min-w-0 flex-1 space-y-1.5">
        <div className="flex flex-wrap items-center gap-2">
          <Stars value={r.rating} />
          {hidden && <Badge className="rounded-md bg-red-100 text-red-800">Disembunyikan</Badge>}
          {r.edited && <span className="text-xs text-muted-foreground">(diubah)</span>}
        </div>
        <p className="text-sm font-medium">
          {r.product.name} <span className="font-normal text-muted-foreground">· {r.storeName}</span>
        </p>
        {r.comment ? (
          <p className="whitespace-pre-line break-words text-sm">{r.comment}</p>
        ) : (
          <p className="text-sm italic text-muted-foreground">Tanpa komentar</p>
        )}
        <p className="text-xs text-muted-foreground">
          {r.reviewer.name} ({r.reviewer.email}) · {formatDateTime(r.createdAt)}
        </p>
      </div>
      <div className="shrink-0">
        {hidden ? (
          <ActionButton
            label="Tampilkan"
            run={() => setReviewStatus(r.id, 'visible')}
            onDone={(saved) => {
              onChange({ ...r, status: saved.status })
              toast.success('Ulasan tampil lagi dan dihitung di rating produk.')
            }}
          />
        ) : (
          <ActionButton
            label="Sembunyikan"
            destructive
            run={() => setReviewStatus(r.id, 'hidden')}
            onDone={(saved) => {
              onChange({ ...r, status: saved.status })
              toast.success('Ulasan disembunyikan.')
            }}
            confirm={{
              title: 'Sembunyikan ulasan ini?',
              action: 'Sembunyikan',
              description: (
                <p>
                  Ulasan bintang {r.rating} untuk “{r.product.name}” hilang dari halaman produk dan tidak lagi dihitung di
                  rating. Penulisnya tetap melihat ulasannya dengan tanda “disembunyikan”. Bisa ditampilkan lagi kapan
                  saja.
                </p>
              ),
            }}
          />
        )}
      </div>
    </article>
  )
}
