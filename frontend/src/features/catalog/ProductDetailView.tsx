import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Info, MapPin, Minus, Plus, Store } from 'lucide-react'
import { ProductImage } from '../../components/product/ProductImage'
import { RatingBadge } from '../../components/shared/RatingBadge'
import { formatCompactCount, formatPriceRange, formatRupiah } from '../../lib/format'
import type { ProductDetail } from '../../types/marketplace'

/**
 * Pure view of one product. Selecting a variant and a quantity is local UI
 * state only; nothing here creates a cart, an order, or reserves stock.
 * The parent renders it with key={product.id} so the selection resets when
 * the product changes.
 */
export function ProductDetailView({ product }: { product: ProductDetail }) {
  const firstAvailable = product.variants.find((v) => v.stock > 0) ?? product.variants[0]
  const [variantId, setVariantId] = useState(firstAvailable.id)
  const [quantity, setQuantity] = useState(1)

  const variant = product.variants.find((v) => v.id === variantId) ?? firstAvailable
  const available = variant.stock > 0
  const hasChoices = product.variants.length > 1

  const chooseVariant = (id: string) => {
    const next = product.variants.find((v) => v.id === id)
    setVariantId(id)
    // Keep the quantity valid for the newly selected variant's stock.
    if (next) setQuantity((q) => Math.max(1, Math.min(q, next.stock || 1)))
  }

  return (
    <article>
      <nav aria-label="Navigasi halaman" className="mb-4 text-sm text-slate-500">
        <Link to="/products" className="hover:text-emerald-700">Produk</Link>
        {' / '}
        <Link to={`/categories/${product.category.slug}`} className="hover:text-emerald-700">
          {product.category.name}
        </Link>
      </nav>

      <div className="grid gap-8 md:grid-cols-2">
        <ProductImage
          id={product.id}
          name={product.name}
          image={product.images[0]}
          className="rounded-xl border border-slate-200"
        />

        <div className="space-y-5">
          <header>
            <h1 className="text-2xl font-semibold text-slate-900">{product.name}</h1>
            <div className="mt-2 flex items-center gap-2 text-sm text-slate-600">
              <RatingBadge rating={product.rating} />
              <span>· {formatCompactCount(product.soldCount)} terjual</span>
            </div>
          </header>

          <p className="text-3xl font-bold text-slate-900">
            {hasChoices ? formatRupiah(variant.price) : formatPriceRange(product.lowestPrice, product.highestPrice)}
          </p>

          {hasChoices && (
            <fieldset>
              <legend className="mb-2 text-sm font-medium text-slate-700">Pilih varian</legend>
              <div className="flex flex-wrap gap-2">
                {product.variants.map((v) => (
                  <label
                    key={v.id}
                    className={`cursor-pointer rounded-md border px-3 py-1.5 text-sm has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-emerald-600 ${
                      v.id === variantId
                        ? 'border-emerald-600 bg-emerald-50 text-emerald-800'
                        : 'border-slate-300 bg-white text-slate-700 hover:border-slate-400'
                    } ${v.stock === 0 ? 'opacity-50' : ''}`}
                  >
                    <input
                      type="radio"
                      name="variant"
                      value={v.id}
                      checked={v.id === variantId}
                      onChange={() => chooseVariant(v.id)}
                      className="sr-only"
                    />
                    {v.name}
                    {v.stock === 0 && ' (habis)'}
                  </label>
                ))}
              </div>
            </fieldset>
          )}

          <p className={`text-sm ${available ? 'text-slate-600' : 'font-medium text-red-600'}`}>
            {available ? `Stok tersedia: ${variant.stock}` : 'Stok habis'}
          </p>

          <div className="flex items-center gap-3">
            <span className="text-sm font-medium text-slate-700" id="qty-label">Jumlah</span>
            <div role="group" aria-labelledby="qty-label" className="inline-flex items-center rounded-md border border-slate-300 bg-white">
              <button
                type="button"
                aria-label="Kurangi jumlah"
                disabled={!available || quantity <= 1}
                onClick={() => setQuantity((q) => q - 1)}
                className="p-2 text-slate-700 disabled:opacity-40"
              >
                <Minus className="size-4" aria-hidden="true" />
              </button>
              <output aria-live="polite" className="min-w-10 text-center text-sm">{quantity}</output>
              <button
                type="button"
                aria-label="Tambah jumlah"
                disabled={!available || quantity >= variant.stock}
                onClick={() => setQuantity((q) => q + 1)}
                className="p-2 text-slate-700 disabled:opacity-40"
              >
                <Plus className="size-4" aria-hidden="true" />
              </button>
            </div>
            {available && (
              <span className="text-sm text-slate-500">Subtotal {formatRupiah(variant.price * quantity)}</span>
            )}
          </div>

          <div className="flex flex-wrap gap-3">
            <button type="button" disabled className="rounded-md bg-emerald-600 px-5 py-2.5 text-sm font-semibold text-white opacity-50">
              Tambah ke keranjang
            </button>
            <button type="button" disabled className="rounded-md border border-emerald-600 px-5 py-2.5 text-sm font-semibold text-emerald-700 opacity-50">
              Beli sekarang
            </button>
          </div>
          <p className="flex items-start gap-2 rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-600">
            <Info className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
            Keranjang dan checkout belum tersedia di prototipe ini. Tombol akan aktif setelah backend selesai.
          </p>

          <Link
            to={`/stores/${product.store.slug}`}
            className="flex items-center gap-3 rounded-xl border border-slate-200 bg-white p-4 hover:border-emerald-400 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
          >
            <span className="flex size-11 items-center justify-center rounded-full bg-emerald-100 text-emerald-700">
              <Store className="size-5" aria-hidden="true" />
            </span>
            <span>
              <span className="block font-medium text-slate-900">{product.store.name}</span>
              <span className="flex items-center gap-1 text-sm text-slate-500">
                <MapPin className="size-3.5" aria-hidden="true" /> {product.store.location}
              </span>
            </span>
          </Link>
        </div>
      </div>

      <section className="mt-10" aria-labelledby="product-description">
        <h2 id="product-description" className="mb-2 text-lg font-semibold text-slate-900">
          Deskripsi produk
        </h2>
        <p className="max-w-3xl whitespace-pre-line text-slate-700">{product.description}</p>
        <p className="mt-3 text-xs text-slate-500">SKU: {variant.sku}</p>
      </section>
    </article>
  )
}
