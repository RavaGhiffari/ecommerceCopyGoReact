import { Link, useSearchParams } from 'react-router-dom'
import { ChevronLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { cn } from '@/lib/utils'
import { Pagination } from '../components/shared/Pagination'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { OrderStatusBadge } from '../features/orders/OrderParts'
import { ORDER_STATUSES, ORDER_STATUS_LABEL, SHIPPING_LABEL } from '../features/orders/status'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatDateTime, formatRupiah } from '../lib/format'
import { listStoreOrders } from '../services/orders'
import type { OrderStatus } from '../types/orders'

const FILTERS: { value: OrderStatus | ''; label: string }[] = [
  { value: '', label: 'Semua' },
  { value: 'paid', label: 'Perlu diproses' },
  ...ORDER_STATUSES.filter((s) => s !== 'paid').map((s) => ({ value: s, label: ORDER_STATUS_LABEL[s] })),
]

/** /seller/orders (behind RequireSeller): orders of the seller's own store. */
export function SellerOrdersPage() {
  usePageTitle('Pesanan toko')
  const [params, setParams] = useSearchParams()
  const raw = params.get('status') ?? ''
  const status = (ORDER_STATUSES as string[]).includes(raw) ? (raw as OrderStatus) : undefined
  const page = Math.max(1, Number(params.get('page')) || 1)
  const { state, reload } = useAsync(() => listStoreOrders(status, page), `store-orders:${status ?? ''}:${page}`)

  const go = (next: { status?: string; page?: number }) => {
    const q: Record<string, string> = {}
    const s = next.status ?? status ?? ''
    if (s) q.status = s
    if (next.page && next.page > 1) q.page = String(next.page)
    setParams(q)
  }

  return (
    <div className="space-y-4">
      <Link to="/seller" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-primary">
        <ChevronLeft className="size-4" aria-hidden="true" /> Toko saya
      </Link>
      <h1 className="text-2xl font-semibold">Pesanan toko</h1>

      <div role="group" aria-label="Filter status" className="flex flex-wrap gap-2">
        {FILTERS.map((f) => {
          const active = (status ?? '') === f.value
          return (
            <Button
              key={f.value || 'all'}
              size="sm"
              variant={active ? 'default' : 'outline'}
              aria-pressed={active}
              onClick={() => go({ status: f.value, page: 1 })}
              className={cn('rounded-full', !active && 'text-muted-foreground')}
            >
              {f.label}
            </Button>
          )
        })}
      </div>

      {state.status === 'loading' && <Skeleton className="h-64 rounded-xl" role="status" aria-label="Memuat pesanan" />}
      {state.status === 'error' && <ErrorState error={state.error} onRetry={reload} title="Gagal memuat pesanan" />}
      {state.status === 'success' && state.data.items.length === 0 && (
        <EmptyState
          title={status ? `Tidak ada pesanan berstatus “${ORDER_STATUS_LABEL[status]}”` : 'Belum ada pesanan'}
          description="Pesanan dari pembeli akan muncul di sini."
        />
      )}
      {state.status === 'success' && state.data.items.length > 0 && (
        <>
          <Card className="py-0">
            <Table>
              <TableHeader className="bg-muted/50">
                <TableRow>
                  <TableHead scope="col" className="px-4">No. pesanan</TableHead>
                  <TableHead scope="col">Penerima</TableHead>
                  <TableHead scope="col">Barang</TableHead>
                  <TableHead scope="col">Kirim</TableHead>
                  <TableHead scope="col" className="text-right">Total</TableHead>
                  <TableHead scope="col">Status</TableHead>
                  <TableHead scope="col" className="px-4"><span className="sr-only">Aksi</span></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {state.data.items.map((o) => (
                  <TableRow key={o.id}>
                    <TableCell className="px-4">
                      <p className="font-medium">{o.orderNumber}</p>
                      <p className="text-xs text-muted-foreground">{formatDateTime(o.createdAt)}</p>
                    </TableCell>
                    <TableCell>
                      {o.shippingAddress.recipientName}
                      <p className="text-xs text-muted-foreground">{o.shippingAddress.city}</p>
                    </TableCell>
                    <TableCell>{o.items.reduce((n, it) => n + it.quantity, 0)}</TableCell>
                    <TableCell>{SHIPPING_LABEL[o.shippingMethod]}</TableCell>
                    <TableCell className="text-right font-medium">{formatRupiah(o.total)}</TableCell>
                    <TableCell>
                      <OrderStatusBadge status={o.status} />
                    </TableCell>
                    <TableCell className="px-4 text-right">
                      <Button variant="outline" size="sm" asChild>
                        <Link to={`/seller/orders/${o.id}`} aria-label={`Detail ${o.orderNumber}`}>
                          Detail
                        </Link>
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>
          <Pagination page={state.data.page} totalPages={state.data.totalPages} onPageChange={(p) => go({ page: p })} />
        </>
      )}
    </div>
  )
}
