// Catalog hooks: the only thing pages and components use to get catalog data.
import { useAsync } from '../../hooks/useAsync'
import { getProduct, getRelatedProducts, getStore, listCategories, listProducts } from '../../services/catalog'
import type { ProductQuery } from '../../types/marketplace'

export function useCategories() {
  return useAsync(() => listCategories(), 'categories')
}

export function useProducts(query: ProductQuery) {
  return useAsync(() => listProducts(query), `products:${JSON.stringify(query)}`)
}

export function useProduct(storeSlug: string, productSlug: string) {
  return useAsync(() => getProduct(storeSlug, productSlug), `product:${storeSlug}/${productSlug}`)
}

export function useRelatedProducts(productId: string) {
  return useAsync(() => getRelatedProducts(productId), `related:${productId}`)
}

export function useStore(slug: string) {
  return useAsync(() => getStore(slug), `store:${slug}`)
}
