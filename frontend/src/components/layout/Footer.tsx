import { Link } from 'react-router-dom'

export function Footer() {
  return (
    <footer className="mt-16 border-t border-slate-200 bg-white">
      <div className="mx-auto flex max-w-6xl flex-col gap-3 px-4 py-8 text-sm text-slate-600 sm:flex-row sm:items-center sm:justify-between">
        <p>
          <span className="font-semibold text-slate-900">PasarKita</span> · prototipe marketplace
          dengan data contoh. Belum ada transaksi sungguhan.
        </p>
        <nav aria-label="Tautan footer" className="flex gap-4">
          <Link to="/products" className="hover:text-emerald-700">
            Semua produk
          </Link>
          <Link to="/status" className="hover:text-emerald-700">
            Status sistem
          </Link>
        </nav>
      </div>
    </footer>
  )
}
