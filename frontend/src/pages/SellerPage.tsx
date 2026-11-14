import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Link } from 'react-router-dom'
import { ExternalLink, Plus } from 'lucide-react'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { useAuth } from '../features/auth/authContext'
import { useCategories } from '../features/catalog/hooks'
import { useMyProducts, useMyStore } from '../features/seller/hooks'
import { OpenStoreForm } from '../features/seller/OpenStoreForm'
import { usePageTitle } from '../hooks/usePageTitle'
import { formatPriceRange } from '../lib/format'

/** /seller: open a store (non-sellers) or manage it (sellers). Behind RequireAuth. */
export function SellerPage() {
  const { user } = useAuth()
  usePageTitle('Toko saya')
  if (!user?.roles.includes('seller')) return <OpenStoreForm />
  return <SellerDashboard />
}

function SellerDashboard() {
  const store = useMyStore()
  const products = useMyProducts()
  const categories = useCategories().state
  const categoryName = (id: string) =>
    categories.status === 'success' ? (categories.data.find((c) => c.id === id)?.name ?? '-') : '…'

  if (store.state.status === 'error') {
    return <ErrorState error={store.state.error} onRetry={store.reload} title="Gagal memuat toko" />
  }

  return (
    <div className="space-y-6">
      {store.state.status === 'loading' ? (
        <Skeleton className="h-24 rounded-xl" role="status" aria-label="Memuat toko" />
      ) : (
        <Card className="flex-row flex-wrap items-start justify-between gap-4 p-5">
          <div>
            <h1 className="text-2xl font-semibold">{store.state.data.name}</h1>
            <p className="text-sm text-muted-foreground">{store.state.data.location}</p>
          </div>
          <Link
            to={`/stores/${store.state.data.slug}`}
            className="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
          >
            Lihat halaman toko <ExternalLink className="size-4" aria-hidden="true" />
          </Link>
        </Card>
      )}

      <section aria-labelledby="my-products">
        <div className="mb-3 flex items-center justify-between">
          <h2 id="my-products" className="text-xl font-semibold">
            Produk
          </h2>
          <Button size="lg" asChild className="font-semibold">
            <Link to="/seller/products/new">
              <Plus aria-hidden="true" /> Tambah produk
            </Link>
          </Button>
        </div>

        {products.state.status === 'loading' && (
          <Skeleton className="h-40 rounded-xl" role="status" aria-label="Memuat produk" />
        )}
        {products.state.status === 'error' && (
          <ErrorState error={products.state.error} onRetry={products.reload} title="Gagal memuat produk" />
        )}
        {products.state.status === 'success' && products.state.data.length === 0 && (
          <EmptyState title="Belum ada produk" description="Tambahkan produk pertamamu agar muncul di katalog." />
        )}
        {products.state.status === 'success' && products.state.data.length > 0 && (
          <Card className="py-0">
            <Table>
              <TableHeader className="bg-muted/50">
                <TableRow>
                  <TableHead scope="col" className="px-4">Produk</TableHead>
                  <TableHead scope="col" className="px-4">Kategori</TableHead>
                  <TableHead scope="col" className="px-4">Harga</TableHead>
                  <TableHead scope="col" className="px-4 text-right">Stok</TableHead>
                  <TableHead scope="col" className="px-4">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {products.state.data.map((p) => (
                  <TableRow key={p.id}>
                    <TableCell className="px-4">
                      <Link to={`/seller/products/${p.id}`} className="font-medium hover:text-primary hover:underline">
                        {p.name}
                      </Link>
                      <span className="block text-xs text-muted-foreground">{p.variantCount} varian</span>
                    </TableCell>
                    <TableCell className="px-4 text-muted-foreground">{categoryName(p.categoryId)}</TableCell>
                    <TableCell className="px-4">{formatPriceRange(p.lowestPrice, p.highestPrice)}</TableCell>
                    <TableCell className={`px-4 text-right ${p.totalStock === 0 ? 'font-medium text-destructive' : ''}`}>{p.totalStock}</TableCell>
                    <TableCell className="px-4">
                      <Badge
                        variant={p.status === 'active' ? 'default' : 'secondary'}
                        className={p.status === 'active' ? 'bg-primary/15 text-primary' : ''}
                      >
                        {p.status === 'active' ? 'Aktif' : 'Diarsipkan'}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>
        )}
      </section>
    </div>
  )
}
