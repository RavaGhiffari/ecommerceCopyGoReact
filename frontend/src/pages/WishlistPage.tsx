import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ProductImage } from '../components/product/ProductImage'
import { Pagination } from '../components/shared/Pagination'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { messageFor } from '../features/auth/serverErrors'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatDateTime, formatRupiah } from '../lib/format'
import { listWishlist, removeFromWishlist } from '../services/wishlist'
import type { WishlistItem } from '../types/wishlist'

const PAGE_SIZE = 12

/** /wishlist (behind RequireAuth): the user's saved products. */
export function WishlistPage() {
  usePageTitle('Favorit saya')
  const [page, setPage] = useState(1)
  const { state, reload, lastData } = useAsync(() => listWishlist(page, PAGE_SIZE), `wishlist:${page}`)
  const [removing, setRemoving] = useState<string | null>(null)

  const remove = async (item: WishlistItem) => {
    setRemoving(item.productId)
    try {
      await removeFromWishlist(item.productId)
      toast.success(`${item.name} dihapus dari favorit.`)
      reload()
    } catch (err) {
      toast.error(messageFor(err))
    } finally {
      setRemoving(null)
    }
  }

  // Keep the previous result on screen while a page change or refresh loads.
  const data = state.status === 'success' ? state.data : lastData

  let body
  if (state.status === 'error') {
    body = <ErrorState error={state.error} onRetry={reload} title="Gagal memuat favorit" />
  } else if (!data) {
    body = (
      <div className="space-y-3" role="status" aria-label="Memuat favorit">
        {Array.from({ length: 3 }, (_, i) => (
          <Skeleton key={i} className="h-24 rounded-xl" />
        ))}
      </div>
    )
  } else if (data.items.length === 0 && data.page === 1) {
    body = (
      <EmptyState
        title="Belum ada favorit"
        description="Tekan ikon hati di halaman produk untuk menyimpannya di sini."
        action={
          <Button asChild>
            <Link to="/products">Jelajahi produk</Link>
          </Button>
        }
      />
    )
  } else {
    body = (
      <div aria-busy={state.status === 'loading'}>
        <Card className="block p-0">
          <ul className="divide-y">
            {data.items.map((it) => (
              <WishlistRow key={it.productId} item={it} busy={removing === it.productId} onRemove={remove} />
            ))}
          </ul>
        </Card>
        <Pagination page={data.page} totalPages={data.totalPages} onPageChange={setPage} />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <div>
        <h1 className="text-2xl font-semibold">Favorit saya</h1>
        {data && data.total > 0 && <p className="text-sm text-muted-foreground">{data.total} produk</p>}
      </div>
      {body}
    </div>
  )
}

function WishlistRow({
  item,
  busy,
  onRemove,
}: {
  item: WishlistItem
  busy: boolean
  onRemove: (item: WishlistItem) => void
}) {
  const href = `/stores/${item.store.slug}/products/${item.slug}`
  return (
    <li className="flex items-center gap-4 p-4">
      {/* ProductImage is w-full: the wrapper sets its size. */}
      <Link to={href} className="w-20 shrink-0" tabIndex={-1} aria-hidden="true">
        <ProductImage id={item.productId} name={item.name} image={item.image ?? undefined} className="rounded-lg" />
      </Link>
      <div className="min-w-0 flex-1">
        <Link to={href} className="line-clamp-2 font-medium hover:text-primary">
          {item.name}
        </Link>
        <Link to={`/stores/${item.store.slug}`} className="text-sm text-muted-foreground hover:text-primary">
          {item.store.name}
        </Link>
        <p className="mt-1 font-semibold">
          <span className="text-xs font-normal text-muted-foreground">Mulai </span>
          {formatRupiah(item.lowestPrice)}
        </p>
        <p className="text-xs text-muted-foreground">Disimpan {formatDateTime(item.addedAt)}</p>
      </div>
      <Button
        variant="ghost"
        size="icon"
        aria-label={`Hapus ${item.name} dari favorit`}
        title="Hapus dari favorit"
        disabled={busy}
        onClick={() => onRemove(item)}
        className="shrink-0 text-muted-foreground hover:text-destructive"
      >
        <Trash2 aria-hidden="true" />
      </Button>
    </li>
  )
}
