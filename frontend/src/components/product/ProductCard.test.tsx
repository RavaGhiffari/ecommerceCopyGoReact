import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { AuthContext } from '../../features/auth/authContext'
import { aProduct, anAuth, plain } from '../../test/fixtures'
import type { ProductSummary } from '../../types/marketplace'
import { ProductCard } from './ProductCard'

function renderCard(product: ProductSummary) {
  // Signed out: the heart button needs no API call.
  return render(
    <AuthContext.Provider value={anAuth()}>
      <MemoryRouter>
        <ProductCard product={product} />
      </MemoryRouter>
    </AuthContext.Provider>,
  )
}

describe('ProductCard', () => {
  it('shows name, price, store, location, rating and sold count, linked to the store-scoped URL', () => {
    renderCard(aProduct())

    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', '/stores/gadget-nusantara/products/kabel-usb-c-braided')
    expect(screen.getByRole('heading', { name: 'Kabel USB-C Braided' })).toBeInTheDocument()
    expect(plain(link.textContent)).toContain('Rp 39.000')
    expect(link).toHaveTextContent('Jakarta Barat · Gadget Nusantara')
    expect(link).toHaveTextContent('21 rb terjual')
    expect(link).toHaveTextContent('4,5')
    expect(screen.queryByText('mulai')).not.toBeInTheDocument()
    expect(screen.queryByText('Stok habis')).not.toBeInTheDocument()
  })

  it('says "mulai" with the lowest price when variants differ', () => {
    renderCard(aProduct({ lowestPrice: '49000.00', highestPrice: '59000.00' }))
    expect(screen.getByText('mulai')).toBeInTheDocument()
    expect(plain(screen.getByRole('link').textContent)).toContain('Rp 49.000')
  })

  it('marks a product without stock', () => {
    renderCard(aProduct({ inStock: false }))
    expect(screen.getByText('Stok habis')).toBeInTheDocument()
  })

  it('keeps the wishlist button outside the link (no button inside <a>)', () => {
    renderCard(aProduct())
    const heart = screen.getByRole('button', { name: /favorit/ })
    expect(screen.getByRole('link')).not.toContainElement(heart)
  })
})
