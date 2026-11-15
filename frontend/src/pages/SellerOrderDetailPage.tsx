import { useState, type FormEvent } from 'react'
import { useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { ErrorState } from '../components/shared/StateViews'
import { FormAlert } from '../components/shared/TextField'
import { messageFor } from '../features/auth/serverErrors'
import { OrderDetailLayout, OrderDetailSkeleton } from '../features/orders/OrderDetailLayout'
import { ORDER_STATUS_LABEL, SELLER_NEXT } from '../features/orders/status'
import { useAsyncResource } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { ApiError } from '../lib/api'
import { getStoreOrder, updateStoreOrderStatus } from '../services/orders'
import { MAX_NOTE_LENGTH, type Order } from '../types/orders'

/** /seller/orders/:id (behind RequireSeller). Another store's order is a 404 from the API. */
export function SellerOrderDetailPage() {
  const { id = '' } = useParams()
  return <StoreOrder key={id} id={id} />
}

function StoreOrder({ id }: { id: string }) {
  usePageTitle('Detail pesanan toko')
  const { state, replace, reload } = useAsyncResource(() => getStoreOrder(id), `store-order:${id}`)

  if (state.status === 'loading') return <OrderDetailSkeleton />
  if (state.status === 'error') return <ErrorState error={state.error} onRetry={reload} title="Gagal memuat pesanan" />

  return (
    <OrderDetailLayout
      order={state.data}
      back={{ to: '/seller/orders', label: 'Pesanan toko' }}
      // key: a new status starts with an empty note field.
      actions={<SellerActions key={state.data.status} order={state.data} onChange={replace} onReload={reload} />}
    />
  )
}

const WAITING_NOTE: Partial<Record<Order['status'], string>> = {
  pending_payment: 'Menunggu pembeli membayar. Pesanan bisa diproses setelah dibayar.',
  delivered: 'Pesanan selesai.',
  cancelled: 'Pesanan dibatalkan; stoknya sudah dikembalikan.',
  refunded: 'Pembeli membatalkan pesanan; stoknya sudah dikembalikan.',
}

function SellerActions({
  order,
  onChange,
  onReload,
}: {
  order: Order
  onChange: (o: Order) => void
  onReload: () => void
}) {
  const next = SELLER_NEXT[order.status]
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!next) {
    return (
      <div className="space-y-2">
        <h2 className="font-semibold">Tindakan</h2>
        <p className="text-sm text-muted-foreground">{WAITING_NOTE[order.status] ?? 'Tidak ada tindakan.'}</p>
      </div>
    )
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const updated = await updateStoreOrderStatus(order.id, {
        status: next.to,
        ...(note.trim() ? { note: note.trim() } : {}),
      })
      onChange(updated)
      toast.success(`Status pesanan: ${ORDER_STATUS_LABEL[updated.status]}.`)
    } catch (err) {
      // e.g. the buyer cancelled a moment ago: show what the order is now.
      if (err instanceof ApiError && err.code === 'INVALID_STATUS_TRANSITION') {
        toast.error('Status pesanan sudah berubah. Halaman dimuat ulang.')
        onReload()
      } else if (err instanceof ApiError && err.fieldErrors().note) {
        setError(`Catatan maksimal ${MAX_NOTE_LENGTH} karakter.`)
      } else {
        setError(messageFor(err))
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={(e) => void submit(e)} className="space-y-3">
      <h2 className="font-semibold">Tindakan</h2>
      <p className="text-sm text-muted-foreground">
        Langkah berikutnya: <span className="font-medium text-foreground">{ORDER_STATUS_LABEL[next.to]}</span>
      </p>
      <div className="space-y-1.5">
        <Label htmlFor="status-note">{next.noteLabel}</Label>
        <Textarea
          id="status-note"
          value={note}
          onChange={(e) => setNote(e.target.value)}
          maxLength={MAX_NOTE_LENGTH}
          rows={2}
          disabled={busy}
          placeholder={next.to === 'shipped' ? 'mis. Resi JNE 0123456789' : undefined}
        />
      </div>
      <FormAlert message={error} />
      <Button type="submit" className="w-full" disabled={busy}>
        {busy ? 'Menyimpan…' : next.action}
      </Button>
    </form>
  )
}
