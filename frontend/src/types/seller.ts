// Mirrors of the seller API (backend/internal/modules/{stores,products}/dto.go).
// Keep the two in sync.
import type { Money, ProductImage, ProductVariant } from './marketplace'

export interface SellerStore {
  id: string
  slug: string
  name: string
  description: string
  location: string
  status: 'active' | 'suspended'
  createdAt: string
}

export type ProductStatus = 'active' | 'archived'

export interface SellerProductSummary {
  id: string
  slug: string
  name: string
  status: ProductStatus
  categoryId: string
  variantCount: number
  totalStock: number
  lowestPrice: Money
  highestPrice: Money
  updatedAt: string
}

export interface SellerProduct {
  id: string
  slug: string
  name: string
  description: string
  status: ProductStatus
  categoryId: string
  images: ProductImage[]
  variants: ProductVariant[]
  createdAt: string
  updatedAt: string
}

export interface OpenStoreInput {
  name: string
  description: string
  location: string
}

export interface VariantInput {
  name: string
  sku: string
  price: Money // decimal string, sent as-is so no precision is lost
  stock: number
}

export interface CreateProductInput {
  name: string
  description: string
  categoryId: string
  images: ProductImage[]
  variants: VariantInput[]
}

export interface UpdateProductInput {
  name?: string
  description?: string
  categoryId?: string
  status?: ProductStatus
  images?: ProductImage[]
}
