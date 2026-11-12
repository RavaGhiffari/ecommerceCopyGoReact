// Catalog hooks: the only thing pages and components use to get catalog data.
// They call src/services, so switching from mocks to the real API never
// touches this file's callers.
import { useAsync } from '../../hooks/useAsync'
import { getProduct, getStore, listCategories, listProducts } from '../../services/catalog'
import { currentSimulation } from '../../services/simulate'
import type { ProductQuery } from '../../types/marketplace'

// The simulation flag is part of every key so changing ?simulate= refetches.
const sim = () => currentSimulation() ?? 'none'

export function useCategories() {
  return useAsync(() => listCategories(), `categories:${sim()}`)
}

export function useProducts(query: ProductQuery) {
  return useAsync(() => listProducts(query), `products:${sim()}:${JSON.stringify(query)}`)
}

export function useProduct(storeSlug: string, productSlug: string) {
  return useAsync(() => getProduct(storeSlug, productSlug), `product:${sim()}:${storeSlug}/${productSlug}`)
}

export function useStore(slug: string) {
  return useAsync(() => getStore(slug), `store:${sim()}:${slug}`)
}
