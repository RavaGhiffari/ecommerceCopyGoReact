import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { LineSkeleton, LoadingRegion, StatTileSkeleton } from '../components/shared/Skeletons'
import { ErrorState } from '../components/shared/StateViews'
import { ORDER_STATUSES, ORDER_STATUS_LABEL, ORDER_STATUS_TONE } from '../features/orders/status'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatCompactCount, formatDateTime, formatRupiah } from '../lib/format'
import { getAdminStats } from '../services/admin'

/** /admin: platform numbers from GET /admin/stats. Nothing is computed here but formatting. */
export function AdminDashboardPage() {
  usePageTitle('Panel admin')
  const { state, reload } = useAsync(() => getAdminStats(), 'admin-stats')

  if (state.status === 'loading') {
    return (
      <LoadingRegion label="Memuat ringkasan platform" className="block space-y-4">
        <Skeleton className="h-8 w-56" />
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }, (_, i) => (
            <StatTileSkeleton key={i} />
          ))}
        </div>
        <Card className="gap-3 p-4 lg:max-w-md">
          <LineSkeleton className="w-32" />
          {Array.from({ length: 7 }, (_, i) => (
            <div key={i} className="flex items-center justify-between">
              <Skeleton className="h-5 w-28 rounded-md" />
              <LineSkeleton className="w-6" />
            </div>
          ))}
        </Card>
      </LoadingRegion>
    )
  }
  if (state.status === 'error') {
    return <ErrorState error={state.error} onRetry={reload} title="Gagal memuat ringkasan platform" />
  }

  const s = state.data
  const n = (v: number | undefined) => formatCompactCount(v ?? 0)

  return (
    <section aria-labelledby="platform-summary" className="space-y-4">
      <div>
        <h1 id="platform-summary" className="text-2xl font-semibold">
          Ringkasan platform
        </h1>
        <p className="text-sm text-muted-foreground">30 hari terakhir dihitung sejak {formatDateTime(s.since)}.</p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <StatTile label="GMV 30 hari" value={formatRupiah(s.gmv.last30Days)}>
          Total: {formatRupiah(s.gmv.allTime)} · tanpa ongkir
        </StatTile>
        <StatTile label="Pengguna aktif" value={n(s.users.active)}>
          <Link to="/admin/users?status=suspended" className="hover:text-primary hover:underline">
            {n(s.users.suspended)} ditangguhkan
          </Link>
        </StatTile>
        <StatTile label="Toko aktif" value={n(s.stores.active)}>
          <Link to="/admin/stores?status=suspended" className="hover:text-primary hover:underline">
            {n(s.stores.suspended)} ditangguhkan
          </Link>
        </StatTile>
        <StatTile label="Produk tampil" value={n(s.publicProducts)}>
          Aktif, toko aktif, punya varian
        </StatTile>
      </div>

      <Card className="gap-3 p-4 lg:max-w-md">
        <h2 className="text-sm font-semibold">Pesanan per status</h2>
        <ul className="space-y-1.5">
          {ORDER_STATUSES.map((st) => (
            <li key={st}>
              <Link
                to={`/admin/orders?status=${st}`}
                className="flex items-center justify-between gap-2 rounded-md px-1 py-0.5 text-sm hover:bg-muted"
              >
                <Badge className={`rounded-md ${ORDER_STATUS_TONE[st]}`}>{ORDER_STATUS_LABEL[st]}</Badge>
                <span className="font-medium tabular-nums">{s.ordersByStatus[st] ?? 0}</span>
              </Link>
            </li>
          ))}
        </ul>
      </Card>
    </section>
  )
}

function StatTile({ label, value, children }: { label: string; value: string; children: ReactNode }) {
  return (
    <Card className="gap-1 p-4">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="truncate text-2xl font-semibold tabular-nums">{value}</p>
      <p className="text-xs text-muted-foreground">{children}</p>
    </Card>
  )
}
