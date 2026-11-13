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
        <div className="h-24 animate-pulse rounded-xl bg-slate-200" role="status" aria-label="Memuat toko" />
      ) : (
        <section className="flex flex-wrap items-start justify-between gap-4 rounded-xl border border-slate-200 bg-white p-5">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">{store.state.data.name}</h1>
            <p className="text-sm text-slate-600">{store.state.data.location}</p>
          </div>
          <Link
            to={`/stores/${store.state.data.slug}`}
            className="inline-flex items-center gap-1 text-sm font-medium text-emerald-700 hover:underline"
          >
            Lihat halaman toko <ExternalLink className="size-4" aria-hidden="true" />
          </Link>
        </section>
      )}

      <section aria-labelledby="my-products">
        <div className="mb-3 flex items-center justify-between">
          <h2 id="my-products" className="text-xl font-semibold text-slate-900">
            Produk
          </h2>
          <Link
            to="/seller/products/new"
            className="inline-flex items-center gap-1 rounded-md bg-emerald-600 px-3 py-2 text-sm font-semibold text-white hover:bg-emerald-700"
          >
            <Plus className="size-4" aria-hidden="true" /> Tambah produk
          </Link>
        </div>

        {products.state.status === 'loading' && (
          <div className="h-40 animate-pulse rounded-xl bg-slate-200" role="status" aria-label="Memuat produk" />
        )}
        {products.state.status === 'error' && (
          <ErrorState error={products.state.error} onRetry={products.reload} title="Gagal memuat produk" />
        )}
        {products.state.status === 'success' && products.state.data.length === 0 && (
          <EmptyState title="Belum ada produk" description="Tambahkan produk pertamamu agar muncul di katalog." />
        )}
        {products.state.status === 'success' && products.state.data.length > 0 && (
          <div className="overflow-x-auto rounded-xl border border-slate-200 bg-white">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
                <tr>
                  <th scope="col" className="px-4 py-2 font-medium">Produk</th>
                  <th scope="col" className="px-4 py-2 font-medium">Kategori</th>
                  <th scope="col" className="px-4 py-2 font-medium">Harga</th>
                  <th scope="col" className="px-4 py-2 font-medium text-right">Stok</th>
                  <th scope="col" className="px-4 py-2 font-medium">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {products.state.data.map((p) => (
                  <tr key={p.id} className="hover:bg-slate-50">
                    <td className="px-4 py-2">
                      <Link to={`/seller/products/${p.id}`} className="font-medium text-slate-900 hover:text-emerald-700 hover:underline">
                        {p.name}
                      </Link>
                      <span className="block text-xs text-slate-500">{p.variantCount} varian</span>
                    </td>
                    <td className="px-4 py-2 text-slate-600">{categoryName(p.categoryId)}</td>
                    <td className="whitespace-nowrap px-4 py-2">{formatPriceRange(p.lowestPrice, p.highestPrice)}</td>
                    <td className={`px-4 py-2 text-right ${p.totalStock === 0 ? 'font-medium text-red-600' : ''}`}>{p.totalStock}</td>
                    <td className="px-4 py-2">
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          p.status === 'active' ? 'bg-emerald-100 text-emerald-800' : 'bg-slate-200 text-slate-700'
                        }`}
                      >
                        {p.status === 'active' ? 'Aktif' : 'Diarsipkan'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  )
}
