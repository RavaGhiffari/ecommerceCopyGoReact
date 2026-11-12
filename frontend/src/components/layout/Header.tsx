import { useState, type FormEvent } from 'react'
import { Link, NavLink, useNavigate, useSearchParams } from 'react-router-dom'
import { Search, ShoppingCart, Store } from 'lucide-react'
import { useCategories } from '../../features/catalog/hooks'

const COMING_SOON = 'Tersedia setelah backend siap'

export function Header() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  // Key the form on the URL's q so back/forward navigation resets the input.
  const urlQuery = params.get('q') ?? ''

  return (
    <header className="sticky top-0 z-30 border-b border-slate-200 bg-white">
      <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-3">
        <Link
          to="/"
          className="shrink-0 text-xl font-bold text-emerald-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
        >
          Pasar<span className="text-slate-900">Kita</span>
        </Link>

        <SearchForm key={urlQuery} initial={urlQuery} onSearch={(q) => navigate(q ? `/products?q=${encodeURIComponent(q)}` : '/products')} />

        <div className="flex shrink-0 items-center gap-1 sm:gap-2">
          <button
            type="button"
            disabled
            title={COMING_SOON}
            aria-label={`Keranjang. ${COMING_SOON}`}
            className="rounded-md p-2 text-slate-400"
          >
            <ShoppingCart className="size-5" aria-hidden="true" />
          </button>
          <button
            type="button"
            disabled
            title={COMING_SOON}
            className="hidden rounded-md border border-slate-200 px-3 py-1.5 text-sm text-slate-400 sm:block"
          >
            Masuk
          </button>
          <button
            type="button"
            disabled
            title={COMING_SOON}
            className="hidden rounded-md bg-slate-200 px-3 py-1.5 text-sm text-slate-500 sm:block"
          >
            Daftar
          </button>
          <span className="hidden items-center gap-1 pl-2 text-sm text-slate-400 md:flex" title={COMING_SOON}>
            <Store className="size-4" aria-hidden="true" /> Buka toko
          </span>
        </div>
      </div>

      <CategoryNav />
    </header>
  )
}

function SearchForm({ initial, onSearch }: { initial: string; onSearch: (q: string) => void }) {
  const [value, setValue] = useState(initial)

  const submit = (e: FormEvent) => {
    e.preventDefault()
    onSearch(value.trim())
  }

  return (
    <form role="search" onSubmit={submit} className="flex min-w-0 flex-1">
      <label htmlFor="site-search" className="sr-only">
        Cari produk
      </label>
      <input
        id="site-search"
        type="search"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="Cari produk di PasarKita"
        className="min-w-0 flex-1 rounded-l-md border border-r-0 border-slate-300 px-3 py-2 text-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
      />
      <button
        type="submit"
        aria-label="Cari"
        className="rounded-r-md bg-emerald-600 px-3 text-white hover:bg-emerald-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600 focus-visible:ring-offset-2"
      >
        <Search className="size-4" aria-hidden="true" />
      </button>
    </form>
  )
}

function CategoryNav() {
  const { state } = useCategories()
  if (state.status !== 'success') return <div className="h-10 border-t border-slate-100" />

  const link = ({ isActive }: { isActive: boolean }) =>
    `whitespace-nowrap rounded-full px-3 py-1 text-sm ${
      isActive ? 'bg-emerald-100 font-medium text-emerald-800' : 'text-slate-600 hover:bg-slate-100'
    }`

  return (
    <nav aria-label="Kategori" className="border-t border-slate-100">
      <ul className="mx-auto flex max-w-6xl gap-1 overflow-x-auto px-4 py-1.5">
        <li>
          <NavLink to="/products" end className={link}>
            Semua
          </NavLink>
        </li>
        {state.data.map((c) => (
          <li key={c.id}>
            <NavLink to={`/categories/${c.slug}`} className={link}>
              {c.name}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}
