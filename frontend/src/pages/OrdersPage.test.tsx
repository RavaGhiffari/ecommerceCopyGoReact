import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../lib/api'
import { listMyOrders } from '../services/orders'
import type { Order } from '../types/orders'
import { OrdersPage } from './OrdersPage'

// The page talks to the API only through services/orders; replace it.
vi.mock('../services/orders', () => ({ listMyOrders: vi.fn() }))
const list = vi.mocked(listMyOrders)

type Page = Awaited<ReturnType<typeof listMyOrders>>
const pageOf = (items: Order[]): Page => ({ items, page: 1, pageSize: 20, total: items.length, totalPages: 1 }) as Page

function anOrder(overrides: Partial<Order> = {}): Order {
  return {
    id: 'o-1',
    checkoutId: 'co-1',
    orderNumber: 'PK-20261117-ABC123',
    status: 'paid',
    store: { id: 's-1', slug: 'gadget-nusantara', name: 'Gadget Nusantara' },
    shippingMethod: 'regular',
    shippingFee: '15000.00',
    itemsSubtotal: '118000.00',
    total: '133000.00',
    items: [
      {
        id: 'oi-1',
        variantId: 'v-1',
        productId: 'p-1',
        productSlug: 'kabel-usb-c-braided',
        productName: 'Kabel USB-C Braided',
        variantName: '2 meter',
        sku: 'GN-KBL-2M',
        unitPrice: '59000.00',
        quantity: 2,
        lineTotal: '118000.00',
        image: null,
      },
    ],
    createdAt: '2026-11-17T03:00:00Z',
    updatedAt: '2026-11-17T03:00:00Z',
    ...overrides,
  } as Order
}

const renderPage = () =>
  render(
    <MemoryRouter>
      <OrdersPage />
    </MemoryRouter>,
  )

// Braces matter: a hook that RETURNS a function (mockReset returns the mock)
// has it called as a teardown, which here would wait on a pending promise.
beforeEach(() => {
  list.mockReset()
})

describe('OrdersPage', () => {
  it('shows a loading region first', () => {
    list.mockReturnValue(new Promise(() => {})) // never settles
    renderPage()
    expect(screen.getByRole('status', { name: 'Memuat pesanan' })).toBeInTheDocument()
  })

  it('shows the empty state', async () => {
    list.mockResolvedValue(pageOf([]))
    renderPage()
    expect(await screen.findByRole('heading', { name: 'Belum ada pesanan' })).toBeInTheDocument()
  })

  it('shows the API error code and message', async () => {
    list.mockRejectedValue(new ApiError(503, 'DATABASE_UNAVAILABLE', 'Database tidak tersedia'))
    renderPage()
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Gagal memuat pesanan')
    expect(alert).toHaveTextContent('DATABASE_UNAVAILABLE')
  })

  it('lists orders; only an unpaid one gets a "Bayar" link to its checkout', async () => {
    list.mockResolvedValue(
      pageOf([anOrder(), anOrder({ id: 'o-2', orderNumber: 'PK-20261117-XYZ789', status: 'pending_payment', checkoutId: 'co-2' })]),
    )
    renderPage()

    const paid = await screen.findByRole('article', { name: 'Pesanan PK-20261117-ABC123' })
    expect(paid).toHaveTextContent('Dibayar')
    expect(paid.querySelector('a[href="/checkout/co-1"]')).toBeNull()

    const unpaid = screen.getByRole('article', { name: 'Pesanan PK-20261117-XYZ789' })
    expect(unpaid.querySelector('a[href="/checkout/co-2"]')).toHaveTextContent('Bayar')
  })
})
