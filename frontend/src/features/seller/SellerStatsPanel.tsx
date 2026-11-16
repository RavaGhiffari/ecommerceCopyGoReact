import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { LineSkeleton, LoadingRegion, StatTileSkeleton } from '../../components/shared/Skeletons'
import { ErrorState } from '../../components/shared/StateViews'
import { formatCompactCount, formatRupiah } from '../../lib/format'
import { ORDER_STATUSES, ORDER_STATUS_LABEL, ORDER_STATUS_TONE } from '../orders/status'
import { useMyStats } from './hooks'

/**
 * Seller dashboard numbers (6c): stat tiles, orders per status and the best
 * sellers of the last 30 days. Every number comes from GET /seller/stats;
 * nothing is computed here except formatting.
 */
export function SellerStatsPanel() {
  const { state, reload } = useMyStats()

  if (state.status === 'loading') {
    return (
      <LoadingRegion label="Memuat ringkasan toko" className="block space-y-4">
        <Skeleton className="h-7 w-48" />
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }, (_, i) => (
            <StatTileSkeleton key={i} />
          ))}
        </div>
        <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
          <Card className="gap-3 p-4">
            <LineSkeleton className="w-32" />
            {Array.from({ length: 7 }, (_, i) => (
              <div key={i} className="flex items-center justify-between">
                <Skeleton className="h-5 w-28 rounded-md" />
                <LineSkeleton className="w-6" />
              </div>
            ))}
          </Card>
          <Card className="gap-3 p-4">
            <LineSkeleton className="w-40" />
            {Array.from({ length: 5 }, (_, i) => (
              <div key={i} className="flex items-center justify-between gap-4 border-t pt-3">
                <LineSkeleton className="w-1/2" />
                <LineSkeleton className="w-8" />
                <LineSkeleton className="w-24" />
              </div>
            ))}
          </Card>
        </div>
      </LoadingRegion>
    )
  }
  if (state.status === 'error') {
    return <ErrorState error={state.error} onRetry={reload} title="Gagal memuat ringkasan toko" />
  }

  const s = state.data
  const toProcess = s.ordersByStatus.paid
  const rating = s.rating.count > 0 ? s.rating.average.toFixed(1).replace('.', ',') : '–'

  return (
    <section aria-labelledby="store-summary" className="space-y-4">
      <h2 id="store-summary" className="text-xl font-semibold">
        Ringkasan toko
      </h2>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <StatTile label="Pendapatan 30 hari" value={formatRupiah(s.revenue.last30Days)}>
          Total: {formatRupiah(s.revenue.allTime)} · tanpa ongkir
        </StatTile>
        <StatTile label="Unit terjual 30 hari" value={formatCompactCount(s.unitsSold.last30Days)}>
          Total: {formatCompactCount(s.unitsSold.allTime)} unit
        </StatTile>
        <StatTile
          label="Perlu diproses"
          value={String(toProcess)}
          highlight={toProcess > 0}
        >
          <Link to="/seller/orders?status=paid" className="font-medium text-primary hover:underline">
            Lihat pesanan dibayar
          </Link>
        </StatTile>
        <StatTile label="Rating toko" value={rating}>
          {s.rating.count > 0 ? `dari ${formatCompactCount(s.rating.count)} ulasan` : 'Belum ada ulasan'}
        </StatTile>
      </div>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
        <Card className="gap-3 p-4">
          <h3 className="text-sm font-semibold">Pesanan per status</h3>
          <ul className="space-y-1.5">
            {ORDER_STATUSES.map((st) => (
              <li key={st}>
                <Link
                  to={`/seller/orders?status=${st}`}
                  className="flex items-center justify-between gap-2 rounded-md px-1 py-0.5 text-sm hover:bg-muted"
                >
                  <Badge className={`rounded-md ${ORDER_STATUS_TONE[st]}`}>{ORDER_STATUS_LABEL[st]}</Badge>
                  <span className="font-medium tabular-nums">{s.ordersByStatus[st]}</span>
                </Link>
              </li>
            ))}
          </ul>
        </Card>

        <Card className="gap-0 py-0">
          <h3 className="p-4 pb-2 text-sm font-semibold">Produk terlaris 30 hari</h3>
          {s.topProducts.length === 0 ? (
            <p className="px-4 pb-4 text-sm text-muted-foreground">Belum ada penjualan dalam 30 hari terakhir.</p>
          ) : (
            <Table>
              <TableHeader className="bg-muted/50">
                <TableRow>
                  <TableHead scope="col" className="px-4">Produk</TableHead>
                  <TableHead scope="col" className="px-4 text-right">Unit</TableHead>
                  <TableHead scope="col" className="px-4 text-right">Pendapatan</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {s.topProducts.map((p) => (
                  <TableRow key={p.productId}>
                    <TableCell className="px-4">
                      <Link to={`/seller/products/${p.productId}`} className="font-medium hover:text-primary hover:underline">
                        {p.name}
                      </Link>
                      {p.status === 'archived' && (
                        <Badge variant="secondary" className="ml-2">
                          Diarsipkan
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="px-4 text-right tabular-nums">{p.units}</TableCell>
                    <TableCell className="px-4 text-right tabular-nums">{formatRupiah(p.revenue)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </Card>
      </div>
    </section>
  )
}

function StatTile({
  label,
  value,
  highlight = false,
  children,
}: {
  label: string
  value: string
  highlight?: boolean
  children: ReactNode
}) {
  return (
    <Card className={`gap-1 p-4 ${highlight ? 'ring-primary/60' : ''}`}>
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className={`truncate text-2xl font-semibold tabular-nums ${highlight ? 'text-primary' : ''}`}>{value}</p>
      <p className="text-xs text-muted-foreground">{children}</p>
    </Card>
  )
}
