import { Card } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Pagination } from '../components/shared/Pagination'
import { TableSkeleton } from '../components/shared/Skeletons'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { ResultCount } from '../features/admin/AdminParts'
import { useAdminListParams } from '../features/admin/hooks'
import { AUDIT_ACTION_LABEL, auditSummary } from '../features/admin/labels'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatDateTime } from '../lib/format'
import { listAuditLogs } from '../services/admin'

const HEADERS = ['Waktu', 'Admin', 'Aksi', 'Target']
const NO_STATUSES: readonly string[] = []

/** /admin/audit-logs: every admin action, newest first (append-only on the server). */
export function AdminAuditLogsPage() {
  usePageTitle('Log audit · Admin')
  const { page, update } = useAdminListParams(NO_STATUSES)
  const { state, reload } = useAsync(() => listAuditLogs(page), `admin-audit:${page}`)

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold">Log audit</h1>
        <p className="text-sm text-muted-foreground">
          Setiap aksi admin dicatat dalam transaksi yang sama dengan aksinya, dan catatannya tidak bisa diubah.
        </p>
      </div>

      {state.status === 'loading' && <TableSkeleton label="Memuat log audit" headers={HEADERS} />}
      {state.status === 'error' && <ErrorState error={state.error} onRetry={reload} title="Gagal memuat log audit" />}
      {state.status === 'success' && state.data.items.length === 0 && (
        <EmptyState title="Belum ada aksi admin" description="Menangguhkan, mengubah kategori, dan moderasi ulasan muncul di sini." />
      )}
      {state.status === 'success' && state.data.items.length > 0 && (
        <>
          <ResultCount total={state.data.total} noun="catatan" />
          <Card className="py-0">
            <Table>
              <TableHeader className="bg-muted/50">
                <TableRow>
                  <TableHead scope="col" className="px-4">Waktu</TableHead>
                  <TableHead scope="col">Admin</TableHead>
                  <TableHead scope="col">Aksi</TableHead>
                  <TableHead scope="col" className="px-4">Target</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {state.data.items.map((log) => (
                  <TableRow key={log.id}>
                    <TableCell className="px-4 text-sm whitespace-nowrap">{formatDateTime(log.createdAt)}</TableCell>
                    <TableCell>
                      {log.actor.name}
                      <p className="text-xs text-muted-foreground">{log.actor.email}</p>
                    </TableCell>
                    <TableCell>
                      <p>{AUDIT_ACTION_LABEL[log.action] ?? log.action}</p>
                      <p className="font-mono text-xs text-muted-foreground">{log.action}</p>
                    </TableCell>
                    <TableCell className="px-4 text-sm">{auditSummary(log)}</TableCell>
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
