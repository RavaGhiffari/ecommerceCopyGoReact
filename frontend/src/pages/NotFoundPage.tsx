import { Link } from 'react-router-dom'
import { EmptyState } from '../components/shared/StateViews'
import { usePageTitle } from '../hooks/usePageTitle'

export function NotFoundPage() {
  usePageTitle('Halaman tidak ditemukan')
  return (
    <EmptyState
      title="Halaman tidak ditemukan"
      description="Alamat yang kamu buka tidak ada atau sudah dipindahkan."
      action={
        <Link
          to="/"
          className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700"
        >
          Kembali ke beranda
        </Link>
      }
    />
  )
}
