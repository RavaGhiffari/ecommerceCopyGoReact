import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { CartContext, type CartContextValue } from '../features/cart/cartContext'
import { ApiError } from '../lib/api'
import { aCart, aCartContext, aCartLine, plain } from '../test/fixtures'
import { CartPage } from './CartPage'

function renderCart(ctx: CartContextValue) {
  return render(
    <CartContext.Provider value={ctx}>
      <MemoryRouter>
        <CartPage />
      </MemoryRouter>
    </CartContext.Provider>,
  )
}

const summary = () => screen.getByRole('complementary', { name: 'Ringkasan belanja' })

describe('CartPage', () => {
  it('shows a loading region while the cart loads', () => {
    renderCart(aCartContext({ status: 'loading' }))
    expect(screen.getByRole('status', { name: 'Memuat keranjang' })).toBeInTheDocument()
  })

  it('shows the error with a retry', async () => {
    const reload = vi.fn()
    renderCart(aCartContext({ status: 'error', error: new ApiError(0, 'NETWORK_ERROR', 'Tidak dapat terhubung'), reload }))
    expect(screen.getByRole('alert')).toHaveTextContent('Gagal memuat keranjang')
    await userEvent.click(screen.getByRole('button', { name: /Coba lagi/ }))
    expect(reload).toHaveBeenCalledOnce()
  })

  it('shows the empty state with a way to shop', () => {
    renderCart(aCartContext({ cart: aCart([], '0.00') }))
    expect(screen.getByRole('heading', { name: 'Keranjangmu masih kosong' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Mulai belanja' })).toHaveAttribute('href', '/products')
  })

  it('shows server totals and lets a clean cart continue to checkout', () => {
    renderCart(aCartContext({ cart: aCart([aCartLine()], '118000.00') }))
    expect(plain(summary().textContent)).toContain('Rp 118.000')
    expect(within(summary()).getByRole('button', { name: 'Lanjut ke checkout' })).toBeEnabled()
  })

  it('flags insufficient stock, blocks checkout and offers the fix', async () => {
    const update = vi.fn(async () => aCart([aCartLine({ quantity: 1 })], '59000.00'))
    const line = aCartLine({ quantity: 2, availableQuantity: 1, issue: 'insufficient_stock' })
    renderCart(aCartContext({ cart: aCart([line], '0.00'), update }))

    expect(screen.getByText(/Stok tersisa 1\. Kurangi jumlahnya\./)).toBeInTheDocument()
    expect(within(summary()).getByRole('note')).toHaveTextContent('tidak dihitung di subtotal')
    expect(within(summary()).getByRole('button', { name: 'Lanjut ke checkout' })).toBeDisabled()

    await userEvent.click(screen.getByRole('button', { name: 'Ubah ke 1' }))
    expect(update).toHaveBeenCalledWith('ci-1', 1)
  })

  it('disables quantity buttons for an unavailable product but still allows removing it', () => {
    const line = aCartLine({ issue: 'unavailable' })
    renderCart(aCartContext({ cart: aCart([line], '0.00') }))

    expect(screen.getByText(/sudah tidak dijual/)).toBeInTheDocument()
    const qty = screen.getByRole('group', { name: 'Jumlah Kabel USB-C Braided' })
    expect(within(qty).getByRole('button', { name: 'Tambah jumlah' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Hapus Kabel USB-C Braided dari keranjang' })).toBeEnabled()
  })

  it('shows a failed update next to the line in Indonesian', async () => {
    const update = vi.fn(async () => {
      throw new ApiError(409, 'QUANTITY_EXCEEDS_STOCK', 'x', { available: 2, inCart: 0 })
    })
    renderCart(aCartContext({ cart: aCart([aCartLine({ availableQuantity: 50 })], '118000.00'), update }))

    await userEvent.click(screen.getByRole('button', { name: 'Tambah jumlah' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Stok tersisa 2.')
  })
})
