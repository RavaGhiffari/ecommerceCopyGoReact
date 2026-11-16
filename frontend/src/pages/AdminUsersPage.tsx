import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
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
import { ACCOUNT_STATUSES, ACCOUNT_STATUS_LABEL, ROLE_LABEL } from '../features/admin/labels'
import { useAuth } from '../features/auth/authContext'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatDateTime } from '../lib/format'
import { listAdminUsers, setUserStatus } from '../services/admin'
import type { AdminUser } from '../types/admin'

const HEADERS = ['Pengguna', 'Peran', 'Bergabung', 'Status', '']

/** /admin/users: search by name or email, filter by status, suspend / reactivate. */
export function AdminUsersPage() {
  usePageTitle('Pengguna · Admin')
  const { user: me } = useAuth()
  const { q, status, page, update } = useAdminListParams(ACCOUNT_STATUSES)
  const key = `admin-users:${q}:${status ?? ''}:${page}`
  const { state, reload } = useAsync(() => listAdminUsers({ q, status, page }), key)
  const { apply, patch } = usePatchedRows<AdminUser>(key)

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Pengguna</h1>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <AdminSearch
          key={q}
          initial={q}
          label="Cari pengguna"
          placeholder="Cari nama atau email"
          onSearch={(next) => update({ q: next })}
        />
        <FilterChips
          label="Filter status"
          value={status}
          options={ACCOUNT_STATUSES.map((s) => ({ value: s, label: ACCOUNT_STATUS_LABEL[s] }))}
          onChange={(s) => update({ status: s })}
        />
      </div>

      {state.status === 'loading' && <TableSkeleton label="Memuat pengguna" headers={HEADERS} />}
      {state.status === 'error' && <ErrorState error={state.error} onRetry={reload} title="Gagal memuat pengguna" />}
      {state.status === 'success' && state.data.items.length === 0 && (
        <EmptyState title="Tidak ada pengguna yang cocok" description="Coba kata kunci atau filter lain." />
      )}
      {state.status === 'success' && state.data.items.length > 0 && (
        <>
          <ResultCount total={state.data.total} noun="pengguna" />
          <Card className="py-0">
            <Table>
              <TableHeader className="bg-muted/50">
                <TableRow>
                  <TableHead scope="col" className="px-4">Pengguna</TableHead>
                  <TableHead scope="col">Peran</TableHead>
                  <TableHead scope="col">Bergabung</TableHead>
                  <TableHead scope="col">Status</TableHead>
                  <TableHead scope="col" className="px-4"><span className="sr-only">Aksi</span></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apply(state.data.items).map((u) => (
                  <TableRow key={u.id}>
                    <TableCell className="px-4">
                      <p className="font-medium">
                        {u.name}
                        {u.id === me?.id && <span className="ml-1 text-xs text-muted-foreground">(kamu)</span>}
                      </p>
                      <p className="text-xs text-muted-foreground">{u.email}</p>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {u.roles.map((r) => (
                          <Badge key={r} variant={r === 'admin' ? 'default' : 'secondary'}>
                            {ROLE_LABEL[r] ?? r}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="text-sm">{formatDateTime(u.createdAt)}</TableCell>
                    <TableCell>
                      <AccountStatusBadge status={u.status} />
                    </TableCell>
                    <TableCell className="px-4 text-right">
                      <UserAction user={u} isSelf={u.id === me?.id} onChange={patch} />
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

function UserAction({ user, isSelf, onChange }: { user: AdminUser; isSelf: boolean; onChange: (u: AdminUser) => void }) {
  // The API refuses these too (409); the UI just doesn't offer them.
  if (isSelf || user.roles.includes('admin')) {
    return <span className="text-xs text-muted-foreground">—</span>
  }
  if (user.status === 'suspended') {
    return (
      <ActionButton
        label="Aktifkan"
        run={() => setUserStatus(user.id, 'active')}
        onDone={(u) => {
          onChange(u)
          toast.success(`${u.name} aktif kembali. Ia perlu masuk lagi.`)
        }}
      />
    )
  }
  return (
    <ActionButton
      label="Tangguhkan"
      destructive
      run={() => setUserStatus(user.id, 'suspended')}
      onDone={(u) => {
        onChange(u)
        toast.success(`${u.name} ditangguhkan.`)
      }}
      confirm={{
        title: `Tangguhkan ${user.name}?`,
        action: 'Tangguhkan',
        description: (
          <>
            <p>
              {user.email} langsung keluar dari semua perangkat dan tidak bisa masuk sampai diaktifkan lagi.
            </p>
            {user.roles.includes('seller') && (
              <p>Tokonya tidak ikut ditangguhkan. Tangguhkan tokonya terpisah di halaman Toko bila perlu.</p>
            )}
          </>
        ),
      }}
    />
  )
}
