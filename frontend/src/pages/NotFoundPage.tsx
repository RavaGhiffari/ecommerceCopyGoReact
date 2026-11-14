import { Button } from '@/components/ui/button'
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
        <Button size="lg" asChild>
          <Link to="/">Kembali ke beranda</Link>
        </Button>
      }
    />
  )
}
