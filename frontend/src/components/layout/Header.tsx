import { useState, type FormEvent } from 'react'
import { Link, NavLink, useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { ChevronDown, Heart, LogOut, Package, Search, ShoppingCart, Store, UserRound } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { useAuth } from '../../features/auth/authContext'
import { useCart } from '../../features/cart/cartContext'
import { useCategories } from '../../features/catalog/hooks'

/** Cart icon with the number of units in the cart (anonymous: /cart asks to log in). */
function CartLink() {
  const { itemCount } = useCart()
  const label = itemCount > 0 ? `Keranjang, ${itemCount} barang` : 'Keranjang'
  return (
    <Button variant="ghost" size="icon-lg" asChild className="relative text-muted-foreground hover:text-primary">
      <Link to="/cart" aria-label={label} title={label}>
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
    </Button>
  )
}

export function Header() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  // Key the form on the URL's q so back/forward navigation resets the input.
  const urlQuery = params.get('q') ?? ''

  return (
    <header className="sticky top-0 z-30 border-b bg-background">
      <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-3">
        <Link
          to="/"
          className="shrink-0 rounded-md text-xl font-bold text-primary outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
        >
          Niaga<span className="text-foreground">Heula</span>
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

function useIsSeller() {
  const { user } = useAuth()
  return user?.roles.includes('seller') ?? false
}

/** "Toko saya" for sellers, "Buka toko" for everyone else (/seller decides what to show). */
function SellerLink() {
  const isSeller = useIsSeller()
  return (
    <Button variant="ghost" asChild className="hidden text-muted-foreground hover:text-primary md:inline-flex">
      <Link to="/seller">
        <Store aria-hidden="true" /> {isSeller ? 'Toko saya' : 'Buka toko'}
      </Link>
    </Button>
  )
}

/**
 * Signed out: "Masuk" / "Daftar". Signed in: an account menu (Radix
 * DropdownMenu: keyboard navigation, focus return and Escape come with it)
 * holding the profile link, the seller entry on small screens, and "Keluar".
 */
function AccountActions() {
  const { status, user, logout } = useAuth()
  const isSeller = useIsSeller()
  const navigate = useNavigate()
  const location = useLocation()
  const [busy, setBusy] = useState(false)

  if (status === 'loading') {
    return <Skeleton className="h-8 w-28" aria-hidden="true" />
  }

  if (user) {
    const signOut = async () => {
      setBusy(true)
      // Leave the current page FIRST. If we cleared the user while still on a
      // protected page (e.g. /account), its RequireAuth guard would redirect
      // to /login?next=/account and override this navigation.
      //
      // flushSync: React Router normally commits a navigation inside
      // startTransition (low priority). setUser(null) after logout is an
      // urgent update, so if the logout request returned first, React
      // rendered the OLD page without a user and the guard redirected to
      // /login. That race showed up in E2E once logout moved into the Radix
      // menu. flushSync commits "/" synchronously, so the order is guaranteed.
      await navigate('/', { flushSync: true })
      try {
        await logout()
      } finally {
        setBusy(false)
      }
    }
    return (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="max-w-44 text-foreground/80">
            <UserRound aria-hidden="true" />
            <span className="truncate">{user.name}</span>
            <ChevronDown className="text-muted-foreground" aria-hidden="true" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuLabel className="font-normal">
            <p className="truncate font-medium text-foreground">{user.name}</p>
            <p className="truncate text-xs text-muted-foreground">{user.email}</p>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem asChild>
            <Link to="/account">
              <UserRound aria-hidden="true" /> Akun saya
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem asChild>
            <Link to="/orders">
              <Package aria-hidden="true" /> Pesanan saya
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem asChild>
            <Link to="/wishlist">
              <Heart aria-hidden="true" /> Favorit saya
            </Link>
          </DropdownMenuItem>
          {/* The header shows the seller link from md up; below that it lives here. */}
          <DropdownMenuItem asChild className="md:hidden">
            <Link to="/seller">
              <Store aria-hidden="true" /> {isSeller ? 'Toko saya' : 'Buka toko'}
            </Link>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" disabled={busy} onSelect={() => void signOut()}>
            <LogOut aria-hidden="true" /> Keluar
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    )
  }

  // Come back to the current page after logging in.
  const next =
    location.pathname === '/login' || location.pathname === '/register'
      ? ''
      : `?next=${encodeURIComponent(location.pathname + location.search)}`

  return (
    <>
      <Button variant="outline" asChild className="border-primary text-primary hover:bg-primary/10 hover:text-primary">
        <Link to={`/login${next}`}>Masuk</Link>
      </Button>
      <Button asChild className="hidden sm:inline-flex">
        <Link to={`/register${next}`}>Daftar</Link>
      </Button>
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
      <Input
        id="site-search"
        type="search"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="Cari produk di NiagaHeula"
        className="h-9 min-w-0 flex-1 rounded-r-none border-r-0"
      />
      <Button type="submit" aria-label="Cari" className="h-9 rounded-l-none px-3">
        <Search aria-hidden="true" />
      </Button>
    </form>
  )
}

function CategoryNav() {
  const { state } = useCategories()
  if (state.status !== 'success') return <div className="h-10 border-t border-border/60" />

  const link = ({ isActive }: { isActive: boolean }) =>
    cn(
      'whitespace-nowrap rounded-full px-3 py-1 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/50',
      isActive ? 'bg-primary/15 font-medium text-primary' : 'text-muted-foreground hover:bg-muted hover:text-foreground',
    )

  return (
    <nav aria-label="Kategori" className="border-t border-border/60">
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
