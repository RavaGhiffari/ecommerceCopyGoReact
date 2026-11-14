import { Link } from 'react-router-dom'

export function Footer() {
  return (
    <footer className="mt-16 border-t bg-card">
      <div className="mx-auto flex max-w-6xl flex-col gap-3 px-4 py-8 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
        <p>
          <span className="font-semibold">PasarKita</span> · proyek belajar marketplace
          dengan data contoh. Belum ada transaksi sungguhan.
        </p>
        <nav aria-label="Tautan footer" className="flex gap-4">
          <Link to="/products" className="hover:text-primary">
            Semua produk
          </Link>
          <Link to="/status" className="hover:text-primary">
            Status sistem
          </Link>
        </nav>
      </div>
    </footer>
  )
}
