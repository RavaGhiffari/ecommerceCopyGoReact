// RequireAuth / RequireSeller / RequireAdmin are UX only (the API enforces
// access), but a wrong redirect strands users, so the routing is pinned here.
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { aUser, anAuth, signedIn } from '../test/fixtures'
import { RequireAdmin } from './admin/RequireAdmin'
import { AuthContext, type AuthContextValue } from './auth/authContext'
import { RequireAuth } from './auth/RequireAuth'
import { RequireSeller } from './seller/RequireSeller'

function WhereAmI() {
  const loc = useLocation()
  return <p data-testid="location">{loc.pathname + loc.search}</p>
}

function renderAt(path: string, auth: AuthContextValue) {
  return render(
    <AuthContext.Provider value={auth}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/account" element={<RequireAuth><p>secret account</p></RequireAuth>} />
          <Route path="/seller" element={<WhereAmI />} />
          <Route path="/seller/products/new" element={<RequireSeller><p>product editor</p></RequireSeller>} />
          <Route path="/admin/users" element={<RequireAdmin><p>admin users</p></RequireAdmin>} />
          <Route path="/login" element={<WhereAmI />} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  )
}

describe('RequireAuth', () => {
  it('shows a placeholder, not the page, while the session is checked', () => {
    renderAt('/account', anAuth({ status: 'loading' }))
    expect(screen.getByRole('status', { name: 'Memeriksa sesi' })).toBeInTheDocument()
    expect(screen.queryByText('secret account')).not.toBeInTheDocument()
  })

  it('sends anonymous visitors to /login with the full path in ?next', () => {
    renderAt('/account?tab=reviews', anAuth())
    expect(screen.getByTestId('location')).toHaveTextContent('/login?next=%2Faccount%3Ftab%3Dreviews')
  })

  it('offers a retry when the session check failed', async () => {
    const refresh = vi.fn()
    renderAt('/account', anAuth({ status: 'error', refresh }))
    expect(screen.getByRole('alert')).toHaveTextContent('Tidak dapat memeriksa sesi login')
    await userEvent.click(screen.getByRole('button', { name: /Coba lagi/ }))
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('renders the page for a signed-in user', () => {
    renderAt('/account', signedIn())
    expect(screen.getByText('secret account')).toBeInTheDocument()
  })
})

describe('RequireSeller', () => {
  it('sends a customer without a store to /seller (to open one)', () => {
    renderAt('/seller/products/new', signedIn(aUser({ roles: ['customer'] })))
    expect(screen.getByTestId('location')).toHaveTextContent('/seller')
    expect(screen.queryByText('product editor')).not.toBeInTheDocument()
  })

  it('lets a seller in', () => {
    renderAt('/seller/products/new', signedIn(aUser({ roles: ['customer', 'seller'] })))
    expect(screen.getByText('product editor')).toBeInTheDocument()
  })

  it('still sends anonymous visitors to login first', () => {
    renderAt('/seller/products/new', anAuth())
    expect(screen.getByTestId('location')).toHaveTextContent('/login?next=%2Fseller%2Fproducts%2Fnew')
  })
})

describe('RequireAdmin', () => {
  it('shows a notice to signed-in non-admins', () => {
    renderAt('/admin/users', signedIn(aUser({ roles: ['customer', 'seller'] })))
    expect(screen.getByRole('heading', { name: 'Halaman khusus admin' })).toBeInTheDocument()
    expect(screen.queryByText('admin users')).not.toBeInTheDocument()
  })

  it('lets an admin in', () => {
    renderAt('/admin/users', signedIn(aUser({ roles: ['admin'] })))
    expect(screen.getByText('admin users')).toBeInTheDocument()
  })
})
