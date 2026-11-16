import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Pagination } from '../components/shared/Pagination'
import { TableSkeleton } from '../components/shared/Skeletons'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { FilterChips, ResultCount } from '../features/admin/AdminParts'
import { useAdminListParams } from '../features/admin/hooks'
import { OrderStatusBadge } from '../features/orders/OrderParts'
import { ORDER_STATUSES, ORDER_STATUS_LABEL } from '../features/orders/status'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatDateTime, formatRupiah } from '../lib/format'
import { listAdminOrders } from '../services/admin'

const HEADERS = ['No. pesanan', 'Toko', 'Penerima', 'Barang', 'Total', 'Status', '']

/** /admin/orders: every store's orders, read-only, filterable by status. */
export function AdminOrdersPage() {
  usePageTitle('Pesanan · Admin')
  const { status, page, update } = useAdminListParams(ORDER_STATUSES)
  const { state, reload } = useAsync(() => listAdminOrders(status, page), `admin-orders:${status ?? ''}:${page}`)

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold">Pesanan platform</h1>
        <p className="text-sm text-muted-foreground">Hanya-baca. Status pesanan diubah oleh penjual dan pembeli.</p>
      </div>
      <FilterChips
        label="Filter status"
        value={status}
        options={ORDER_STATUSES.map((s) => ({ value: s, label: ORDER_STATUS_LABEL[s] }))}
        onChange={(s) => update({ status: s })}
      />

      {state.status === 'loading' && <TableSkeleton label="Memuat pesanan" headers={HEADERS} />}
      {state.status === 'error' && <ErrorState error={state.error} onRetry={reload} title="Gagal memuat pesanan" />}
      {state.status === 'success' && state.data.items.length === 0 && (
        <EmptyState title={status ? `Tidak ada pesanan berstatus “${ORDER_STATUS_LABEL[status]}”` : 'Belum ada pesanan'} />
      )}
      {state.status === 'success' && state.data.items.length > 0 && (
        <>
          <ResultCount total={state.data.total} noun="pesanan" />
          <Card className="py-0">
            <Table>
              <TableHeader className="bg-muted/50">
                <TableRow>
                  <TableHead scope="col" className="px-4">No. pesanan</TableHead>
                  <TableHead scope="col">Toko</TableHead>
                  <TableHead scope="col">Penerima</TableHead>
                  <TableHead scope="col" className="text-right">Barang</TableHead>
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
                    <TableCell>{o.store.name}</TableCell>
                    <TableCell>
                      {o.shippingAddress.recipientName}
                      <p className="text-xs text-muted-foreground">{o.shippingAddress.city}</p>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {o.items.reduce((n, it) => n + it.quantity, 0)}
                    </TableCell>
                    <TableCell className="text-right font-medium tabular-nums">{formatRupiah(o.total)}</TableCell>
                    <TableCell>
                      <OrderStatusBadge status={o.status} />
                    </TableCell>
                    <TableCell className="px-4 text-right">
                      <Button variant="outline" size="sm" asChild>
                        <Link to={`/admin/orders/${o.id}`} aria-label={`Detail ${o.orderNumber}`}>
                          Detail
                        </Link>
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
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
