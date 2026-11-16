import { NavLink, Outlet } from 'react-router-dom'
import { cn } from '@/lib/utils'

const SECTIONS = [
  { to: '/admin', label: 'Ringkasan', end: true },
  { to: '/admin/users', label: 'Pengguna' },
  { to: '/admin/stores', label: 'Toko' },
  { to: '/admin/categories', label: 'Kategori' },
  { to: '/admin/orders', label: 'Pesanan' },
  { to: '/admin/reviews', label: 'Ulasan' },
  { to: '/admin/audit-logs', label: 'Log audit' },
]

/** Shell of every /admin page: a section bar above the page (behind RequireAdmin in the router). */
export function AdminLayout() {
  return (
    <div className="space-y-5">
      <div className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-primary">Panel admin</p>
        {/* Scrolls sideways on narrow screens instead of widening the page. */}
        <nav aria-label="Bagian admin" className="-mx-4 overflow-x-auto px-4">
          <ul className="flex w-max gap-1 border-b">
            {SECTIONS.map((s) => (
              <li key={s.to}>
                <NavLink
                  to={s.to}
                  end={s.end}
                  className={({ isActive }) =>
                    cn(
                      '-mb-px inline-block border-b-2 px-3 py-2 text-sm whitespace-nowrap transition-colors',
                      isActive
                        ? 'border-primary font-medium text-primary'
                        : 'border-transparent text-muted-foreground hover:text-foreground',
                    )
                  }
                >
                  {s.label}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
      </div>
      <Outlet />
    </div>
  )
}
