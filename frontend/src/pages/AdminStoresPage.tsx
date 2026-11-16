import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { Card } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Pagination } from '../components/shared/Pagination'
import { TableSkeleton } from '../components/shared/Skeletons'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import {
  AccountStatusBadge,
  ActionButton,
  AdminSearch,
  FilterChips,
  ResultCount,
} from '../features/admin/AdminParts'
import { useAdminListParams, usePatchedRows } from '../features/admin/hooks'
import { ACCOUNT_STATUSES, ACCOUNT_STATUS_LABEL } from '../features/admin/labels'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatDateTime } from '../lib/format'
import { listAdminStores, setStoreStatus } from '../services/admin'
import type { AdminStore } from '../types/admin'

const HEADERS = ['Toko', 'Pemilik', 'Produk aktif', 'Dibuka', 'Status', '']

/** /admin/stores: search by store name, slug or owner email; suspend / reactivate. */
export function AdminStoresPage() {
  usePageTitle('Toko · Admin')
  const { q, status, page, update } = useAdminListParams(ACCOUNT_STATUSES)
  const key = `admin-stores:${q}:${status ?? ''}:${page}`
  const { state, reload } = useAsync(() => listAdminStores({ q, status, page }), key)
  const { apply, patch } = usePatchedRows<AdminStore>(key)

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Toko</h1>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <AdminSearch
          key={q}
          initial={q}
          label="Cari toko"
          placeholder="Cari nama toko, slug, atau email pemilik"
          onSearch={(next) => update({ q: next })}
        />
        <FilterChips
          label="Filter status"
          value={status}
          options={ACCOUNT_STATUSES.map((s) => ({ value: s, label: ACCOUNT_STATUS_LABEL[s] }))}
          onChange={(s) => update({ status: s })}
        />
      </div>

      {state.status === 'loading' && <TableSkeleton label="Memuat toko" headers={HEADERS} />}
      {state.status === 'error' && <ErrorState error={state.error} onRetry={reload} title="Gagal memuat toko" />}
      {state.status === 'success' && state.data.items.length === 0 && (
        <EmptyState title="Tidak ada toko yang cocok" description="Coba kata kunci atau filter lain." />
      )}
      {state.status === 'success' && state.data.items.length > 0 && (
        <>
          <ResultCount total={state.data.total} noun="toko" />
          <Card className="py-0">
            <Table>
              <TableHeader className="bg-muted/50">
                <TableRow>
                  <TableHead scope="col" className="px-4">Toko</TableHead>
                  <TableHead scope="col">Pemilik</TableHead>
                  <TableHead scope="col" className="text-right">Produk aktif</TableHead>
                  <TableHead scope="col">Dibuka</TableHead>
                  <TableHead scope="col">Status</TableHead>
                  <TableHead scope="col" className="px-4"><span className="sr-only">Aksi</span></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apply(state.data.items).map((s) => (
                  <TableRow key={s.id}>
                    <TableCell className="px-4">
                      {/* A suspended store's public page is a 404, so only active ones link. */}
                      {s.status === 'active' ? (
                        <Link to={`/stores/${s.slug}`} className="font-medium hover:text-primary hover:underline">
                          {s.name}
                        </Link>
                      ) : (
                        <p className="font-medium">{s.name}</p>
                      )}
                      <p className="text-xs text-muted-foreground">
                        {s.slug} · {s.location}
                      </p>
                    </TableCell>
                    <TableCell>
                      {s.owner.name}
                      <p className="text-xs text-muted-foreground">{s.owner.email}</p>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">{s.activeProductCount}</TableCell>
                    <TableCell className="text-sm">{formatDateTime(s.createdAt)}</TableCell>
                    <TableCell>
                      <AccountStatusBadge status={s.status} />
                    </TableCell>
                    <TableCell className="px-4 text-right">
                      <StoreAction store={s} onChange={patch} />
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

function StoreAction({ store, onChange }: { store: AdminStore; onChange: (s: AdminStore) => void }) {
  if (store.status === 'suspended') {
    return (
      <ActionButton
        label="Aktifkan"
        run={() => setStoreStatus(store.id, 'active')}
        onDone={(s) => {
          onChange(s)
          toast.success(`${s.name} aktif kembali; produknya tampil lagi di katalog.`)
        }}
      />
    )
  }
  return (
    <ActionButton
      label="Tangguhkan"
      destructive
      run={() => setStoreStatus(store.id, 'suspended')}
      onDone={(s) => {
        onChange(s)
        toast.success(`${s.name} ditangguhkan.`)
      }}
      confirm={{
        title: `Tangguhkan toko ${store.name}?`,
        action: 'Tangguhkan toko',
        description: (
          <>
            <p>
              {store.activeProductCount} produk aktifnya hilang dari katalog dan tidak bisa dibeli. Barang yang sudah di
              keranjang pembeli ditandai tidak tersedia.
            </p>
            <p>Pesanan yang sudah ada tetap berjalan: penjual masih bisa memprosesnya dan pembeli masih bisa membatalkan.</p>
          </>
        ),
      }}
    />
  )
}
