import { useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Star } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ProductImage } from '../../components/product/ProductImage'
import { Pagination } from '../../components/shared/Pagination'
import { ErrorState } from '../../components/shared/StateViews'
import { Stars } from '../../components/shared/Stars'
import { useAsync, type AsyncState } from '../../hooks/useAsync'
import { formatDateTime } from '../../lib/format'
import { listMyReviews, listPendingReviews } from '../../services/reviews'
import type { Paginated } from '../../types/marketplace'
import type { MyReview, PendingReviewItem, ReviewItemRef } from '../../types/reviews'
import { ReviewDialog, type ExistingReview } from './ReviewDialog'

const PAGE_SIZE = 5

type Editing = { item: ReviewItemRef; existing?: ExistingReview }

/** Two cards on /account: items waiting for a review, and the user's reviews. */
export function AccountReviews() {
  const [pendingPage, setPendingPage] = useState(1)
  const [minePage, setMinePage] = useState(1)
  const pending = useAsync(() => listPendingReviews(pendingPage, PAGE_SIZE), `pending-reviews:${pendingPage}`)
  const mine = useAsync(() => listMyReviews(minePage, PAGE_SIZE), `my-reviews:${minePage}`)
  const [editing, setEditing] = useState<Editing | null>(null)

  const saved = () => {
    toast.success(editing?.existing ? 'Ulasan diperbarui.' : 'Terima kasih! Ulasanmu sudah tersimpan.')
    setEditing(null)
    pending.reload()
    mine.reload()
  }

  return (
    <>
      <Card className="block p-5">
        <h2 className="mb-1 text-lg font-semibold">Menunggu ulasan</h2>
        <p className="mb-3 text-sm text-muted-foreground">Barang dari pesanan yang sudah kamu terima.</p>
        <PagedList
          async={pending}
          empty="Tidak ada barang yang menunggu ulasan."
          onPage={setPendingPage}
          render={(it: PendingReviewItem) => (
            <ItemRow
              key={it.orderItemId}
              item={it}
              meta={`${it.storeName} · diterima ${formatDateTime(it.deliveredAt)}`}
              action={
                <Button size="sm" onClick={() => setEditing({ item: it })}>
                  <Star aria-hidden="true" /> Beri ulasan
                </Button>
              }
            />
          )}
        />
      </Card>

      <Card className="block p-5">
        <h2 className="mb-3 text-lg font-semibold">Ulasan saya</h2>
        <PagedList
          async={mine}
          empty="Kamu belum menulis ulasan."
          onPage={setMinePage}
          render={(r: MyReview) => (
            <ItemRow
              key={r.id}
              item={r.item}
              meta={formatDateTime(r.createdAt)}
              action={
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setEditing({ item: r.item, existing: { id: r.id, rating: r.rating, comment: r.comment } })}
                >
                  Ubah
                </Button>
              }
            >
              <div className="mt-1 flex flex-wrap items-center gap-2">
                <Stars value={r.rating} />
                {r.status === 'hidden' && (
                  <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                    Disembunyikan oleh admin
                  </span>
                )}
              </div>
              {r.comment && <p className="mt-1 line-clamp-2 text-sm whitespace-pre-line">{r.comment}</p>}
            </ItemRow>
          )}
        />
      </Card>

      {editing && (
        <ReviewDialog
          open
          onOpenChange={(open) => !open && setEditing(null)}
          target={{
            orderItemId: editing.item.orderItemId,
            productName: editing.item.productName,
            variantName: editing.item.variantName,
          }}
          existing={editing.existing}
          onSaved={saved}
        />
      )}
    </>
  )
}

function PagedList<T>({
  async,
  empty,
  onPage,
  render,
}: {
  async: { state: AsyncState<Paginated<T>>; reload: () => void; lastData?: Paginated<T> }
  empty: string
  onPage: (page: number) => void
  render: (item: T) => ReactNode
}) {
  const { state, reload, lastData } = async
  if (state.status === 'error') return <ErrorState error={state.error} onRetry={reload} title="Gagal memuat ulasan" />
  // Keep the previous page on screen while the next one (or a refresh) loads.
  const data = state.status === 'success' ? state.data : lastData
  if (!data) {
    return (
      <div className="space-y-2" role="status" aria-label="Memuat">
        <Skeleton className="h-16" />
        <Skeleton className="h-16" />
      </div>
    )
  }
  if (data.items.length === 0) return <p className="text-sm text-muted-foreground">{empty}</p>
  return (
    <div aria-busy={state.status === 'loading'}>
      <ul className="divide-y">{data.items.map(render)}</ul>
      <Pagination page={data.page} totalPages={data.totalPages} onPageChange={onPage} />
    </div>
  )
}

function ItemRow({
  item,
  meta,
  action,
  children,
}: {
  item: ReviewItemRef
  meta: string
  action: ReactNode
  children?: ReactNode
}) {
  return (
    <li className="flex gap-3 py-3">
      {/* ProductImage is w-full: the wrapper sets its size. */}
      <div className="w-14 shrink-0">
        <ProductImage id={item.productId} name={item.productName} image={item.image ?? undefined} className="rounded-md" />
      </div>
      <div className="min-w-0 flex-1">
        <Link
          to={`/stores/${item.storeSlug}/products/${item.productSlug}`}
          className="line-clamp-1 text-sm font-medium hover:text-primary"
        >
          {item.productName}
        </Link>
        <p className="text-xs text-muted-foreground">
          {item.variantName} ·{' '}
          <Link to={`/orders/${item.orderId}`} className="hover:text-primary">
            {item.orderNumber}
          </Link>{' '}
          · {meta}
        </p>
        {children}
      </div>
      <div className="shrink-0 self-center">{action}</div>
    </li>
  )
}
