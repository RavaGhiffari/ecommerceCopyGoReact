import { useAsync } from '../../hooks/useAsync'
import { getMyProduct, getMyStore, listMyProducts } from '../../services/seller'

export function useMyStore() {
  return useAsync(() => getMyStore(), 'seller-store')
}

export function useMyProducts() {
  return useAsync(() => listMyProducts(), 'seller-products')
}

export function useMyProduct(id: string) {
  return useAsync(() => getMyProduct(id), `seller-product:${id}`)
}
