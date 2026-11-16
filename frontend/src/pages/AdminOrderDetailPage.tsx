import { useParams } from 'react-router-dom'
import { ErrorState } from '../components/shared/StateViews'
import { OrderDetailLayout, OrderDetailSkeleton } from '../features/orders/OrderDetailLayout'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { getAdminOrder } from '../services/admin'

/** /admin/orders/:id: any order, read-only. Same layout as the buyer and seller pages. */
export function AdminOrderDetailPage() {
  const { id = '' } = useParams()
  usePageTitle('Detail pesanan · Admin')
  const { state, reload } = useAsync(() => getAdminOrder(id), `admin-order:${id}`)

  if (state.status === 'loading') return <OrderDetailSkeleton />
  if (state.status === 'error') return <ErrorState error={state.error} onRetry={reload} title="Gagal memuat pesanan" />

  return (
    <OrderDetailLayout
      order={state.data}
      back={{ to: '/admin/orders', label: 'Pesanan platform' }}
      actions={
        <div className="space-y-2">
          <h2 className="font-semibold">Tindakan</h2>
          <p className="text-sm text-muted-foreground">
            Admin hanya bisa melihat pesanan. Penjual memproses dan mengirim; pembeli bisa membatalkan pesanan yang sudah
            dibayar.
          </p>
        </div>
      }
    />
  )
}
