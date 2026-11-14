import { useState } from 'react'
import { MapPin, Pencil, Plus, Star, Trash2 } from 'lucide-react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ErrorState } from '../../components/shared/StateViews'
import { FormAlert } from '../../components/shared/TextField'
import { useAsync } from '../../hooks/useAsync'
import { deleteAddress, listAddresses, setDefaultAddress } from '../../services/addresses'
import { MAX_ADDRESSES, type Address } from '../../types/address'
import { messageFor } from '../auth/serverErrors'
import { AddressFormDialog } from './AddressFormDialog'

/** "Alamat pengiriman" section of /account: list, add, edit, delete, set default. */
export function AddressBook() {
  const { state, reload } = useAsync(() => listAddresses(), 'addresses')
  // undefined = dialog closed; null = creating; an Address = editing it.
  const [editing, setEditing] = useState<Address | null | undefined>(undefined)
  const [actionError, setActionError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const list = state.status === 'success' ? state.data : []
  const full = list.length >= MAX_ADDRESSES

  // Every change reloads the list: the default flag may move between rows.
  const run = async (id: string, action: () => Promise<unknown>) => {
    setActionError(null)
    setBusyId(id)
    try {
      await action()
      reload()
    } catch (err) {
      setActionError(messageFor(err))
    } finally {
      setBusyId(null)
    }
  }

  return (
    <Card className="block p-5">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="text-lg font-semibold">Alamat pengiriman</h2>
          {state.status === 'success' && (
            <p className="text-sm text-muted-foreground">
              {list.length} dari {MAX_ADDRESSES} alamat
            </p>
          )}
        </div>
        <Button
          variant="outline"
          onClick={() => setEditing(null)}
          disabled={state.status !== 'success' || full}
          title={full ? `Maksimal ${MAX_ADDRESSES} alamat` : undefined}
        >
          <Plus aria-hidden="true" /> Tambah alamat
        </Button>
      </div>

      <div className="space-y-3">
        <FormAlert message={actionError} />

        {state.status === 'loading' && (
          <div className="space-y-3" aria-busy="true" aria-label="Memuat alamat">
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-full" />
          </div>
        )}

        {state.status === 'error' && <ErrorState error={state.error} onRetry={reload} title="Gagal memuat alamat" />}

        {state.status === 'success' && list.length === 0 && (
          <div className="flex flex-col items-center rounded-lg border border-dashed px-6 py-8 text-center">
            <MapPin className="size-8 text-muted-foreground" aria-hidden="true" />
            <p className="mt-2 font-medium">Belum ada alamat</p>
            <p className="text-sm text-muted-foreground">Tambahkan alamat untuk mempercepat checkout.</p>
          </div>
        )}

        {state.status === 'success' && list.length > 0 && (
          <ul className="space-y-3">
            {list.map((a) => (
              <li key={a.id}>
                <AddressItem
                  address={a}
                  busy={busyId === a.id}
                  onEdit={() => setEditing(a)}
                  onSetDefault={() => run(a.id, () => setDefaultAddress(a.id))}
                  onDelete={() => run(a.id, () => deleteAddress(a.id))}
                />
              </li>
            ))}
          </ul>
        )}
      </div>

      <AddressFormDialog
        open={editing !== undefined}
        onOpenChange={(open) => !open && setEditing(undefined)}
        address={editing ?? undefined}
        isFirst={list.length === 0}
        onSaved={() => {
          setEditing(undefined)
          setActionError(null)
          reload()
        }}
      />
    </Card>
  )
}

function AddressItem({
  address: a,
  busy,
  onEdit,
  onSetDefault,
  onDelete,
}: {
  address: Address
  busy: boolean
  onEdit: () => void
  onSetDefault: () => void
  onDelete: () => void
}) {
  return (
    <article
      aria-label={`Alamat ${a.label}`}
      className={`rounded-lg border p-4 ${a.isDefault ? 'border-primary/50 bg-primary/5' : ''}`}
    >
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="font-semibold">{a.label}</h3>
        {a.isDefault && <Badge>Utama</Badge>}
      </div>
      <p className="mt-1 text-sm">
        {a.recipientName} · <span className="text-muted-foreground">{a.phone}</span>
      </p>
      <p className="text-sm text-muted-foreground">
        {a.addressLine}, {a.city}, {a.province} {a.postalCode}
      </p>

      <div className="mt-3 flex flex-wrap gap-2">
        <Button variant="ghost" size="sm" onClick={onEdit} disabled={busy}>
          <Pencil aria-hidden="true" /> Ubah
        </Button>
        {!a.isDefault && (
          <Button variant="ghost" size="sm" onClick={onSetDefault} disabled={busy}>
            <Star aria-hidden="true" /> Jadikan utama
          </Button>
        )}
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" disabled={busy}>
              <Trash2 aria-hidden="true" /> Hapus
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Hapus alamat “{a.label}”?</AlertDialogTitle>
              <AlertDialogDescription>
                Pesanan yang sudah dibuat tidak terpengaruh.
                {a.isDefault && ' Alamat terbaru lainnya akan menjadi alamat utama.'}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Batal</AlertDialogCancel>
              <AlertDialogAction variant="destructive" onClick={onDelete}>
                Hapus
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </article>
  )
}
