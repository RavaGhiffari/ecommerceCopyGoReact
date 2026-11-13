import { useState, type FormEvent } from 'react'
import { Link, NavLink, useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { LogOut, Search, ShoppingCart, Store, UserRound } from 'lucide-react'
import { useAuth } from '../../features/auth/authContext'
import { useCart } from '../../features/cart/cartContext'
import { useCategories } from '../../features/catalog/hooks'

/** Cart icon with the number of units in the cart (anonymous: /cart asks to log in). */
function CartLink() {
  const { itemCount } = useCart()
  const label = itemCount > 0 ? `Keranjang, ${itemCount} barang` : 'Keranjang'
  return (
    <Link to="/cart" aria-label={label} title={label} className="relative rounded-md p-2 text-slate-600 hover:bg-slate-100 hover:text-emerald-700">
      <ShoppingCart className="size-5" aria-hidden="true" />
      {itemCount > 0 && (
        <span
          aria-hidden="true"
          className="absolute -right-0.5 -top-0.5 min-w-5 rounded-full bg-red-600 px-1 text-center text-[11px] font-semibold leading-5 text-white"
        >
          {itemCount > 99 ? '99+' : itemCount}
        </span>
      )}
    </Link>
  )
}

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
          <CartLink />
          <AccountActions />
          <SellerLink />
        </div>
      </div>

      <CategoryNav />
    </header>
  )
}

/** "Toko saya" for sellers, "Buka toko" for everyone else (/seller decides what to show). */
function SellerLink() {
  const { user } = useAuth()
  const isSeller = user?.roles.includes('seller') ?? false
  return (
    <Link
      to="/seller"
      className="hidden items-center gap-1 rounded-md px-2 py-1.5 text-sm text-slate-600 hover:bg-slate-100 hover:text-emerald-700 md:flex"
    >
      <Store className="size-4" aria-hidden="true" /> {isSeller ? 'Toko saya' : 'Buka toko'}
    </Link>
  )
}

/** Login / register links, or the signed-in user's name and a logout button. */
function AccountActions() {
  const { status, user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [busy, setBusy] = useState(false)

  if (status === 'loading') {
    return <span className="h-8 w-28 animate-pulse rounded-md bg-slate-100" aria-hidden="true" />
  }

  if (user) {
    const signOut = async () => {
      setBusy(true)
      // Leave the current page FIRST. If we cleared the user while still on a
      // protected page (e.g. /account), its RequireAuth guard would redirect
      // to /login?next=/account and override this navigation.
      navigate('/')
      try {
        await logout()
      } finally {
        setBusy(false)
      }
    }
    return (
      <div className="flex items-center gap-1">
        <Link
          to="/account"
          className="flex max-w-40 items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-slate-700 hover:bg-slate-100"
        >
          <UserRound className="size-4 shrink-0" aria-hidden="true" />
          <span className="truncate">{user.name}</span>
        </Link>
        <button
          type="button"
          onClick={signOut}
          disabled={busy}
          aria-label="Keluar"
          title="Keluar"
          className="rounded-md p-2 text-slate-600 hover:bg-slate-100 disabled:opacity-50"
        >
          <LogOut className="size-4" aria-hidden="true" />
        </button>
      </div>
    )
  }

  // Come back to the current page after logging in.
  const next =
    location.pathname === '/login' || location.pathname === '/register'
      ? ''
      : `?next=${encodeURIComponent(location.pathname + location.search)}`

  return (
    <>
      <Link
        to={`/login${next}`}
        className="rounded-md border border-emerald-600 px-3 py-1.5 text-sm font-medium text-emerald-700 hover:bg-emerald-50"
      >
        Masuk
      </Link>
      <Link
        to={`/register${next}`}
        className="hidden rounded-md bg-emerald-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-emerald-700 sm:block"
      >
        Daftar
      </Link>
    </>
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
        {state.data.filter((c) => c.parentId === null).map((c) => (
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
