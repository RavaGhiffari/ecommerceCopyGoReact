import { useAsync } from '../../hooks/useAsync'
import { getMyProduct, getMyStats, getMyStore, listMyProducts } from '../../services/seller'

export function useMyStore() {
  return useAsync(() => getMyStore(), 'seller-store')
}

export function useMyProducts() {
  return useAsync(() => listMyProducts(), 'seller-products')
}

export function useMyProduct(id: string) {
  return useAsync(() => getMyProduct(id), `seller-product:${id}`)
}

export function useMyStats() {
  return useAsync(() => getMyStats(), 'seller-stats')
}
