import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { Info, MapPin, Minus, Plus, Store } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { ProductImage } from '../../components/product/ProductImage'
import { RatingBadge } from '../../components/shared/RatingBadge'
import { formatCompactCount, formatPriceRange, formatRupiah } from '../../lib/format'
import type { ProductDetail } from '../../types/marketplace'
import { useAuth } from '../auth/authContext'
import { messageFor } from '../auth/serverErrors'
import { useCart } from '../cart/cartContext'

/**
 * One product: pick a variant and a quantity, then add it to the cart.
 * Adding to the cart never reserves stock (the server re-checks at checkout).
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
    setCartMessage(null)
    // Keep the quantity valid for the newly selected variant's stock.
    if (next) setQuantity((q) => Math.max(1, Math.min(q, next.stock || 1)))
  }

  // Signed-out visitors go to login first and come back here afterwards.
  const { user } = useAuth()
  const { add } = useCart()
  const navigate = useNavigate()
  const location = useLocation()
  const [adding, setAdding] = useState(false)
  const [cartMessage, setCartMessage] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const addToCart = async () => {
    if (!user) {
      navigate(`/login?next=${encodeURIComponent(location.pathname)}`)
      return
    }
    setAdding(true)
    setCartMessage(null)
    try {
      await add(variant.id, quantity)
      setCartMessage({ kind: 'ok', text: `${quantity} × ${variant.name} ditambahkan ke keranjang.` })
    } catch (err) {
      setCartMessage({ kind: 'error', text: messageFor(err) })
    } finally {
      setAdding(false)
    }
  }

  return (
    <article>
      <nav aria-label="Navigasi halaman" className="mb-4 text-sm text-muted-foreground">
        <Link to="/products" className="hover:text-primary">Produk</Link>
        {' / '}
        <Link to={`/categories/${product.category.slug}`} className="hover:text-primary">
          {product.category.name}
        </Link>
      </nav>

      <div className="grid gap-8 md:grid-cols-2">
        <div className="space-y-3">
          <ProductImage
            id={product.id}
            name={product.name}
            image={product.images[0]}
            className="rounded-xl border"
          />
          {product.images.length > 1 && (
            <ul className="grid grid-cols-5 gap-2">
              {product.images.slice(1).map((img) => (
                <li key={img.url}>
                  <img src={img.url} alt={img.alt} loading="lazy" className="aspect-square w-full rounded-md border object-cover" />
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="space-y-5">
          <header>
            <h1 className="text-2xl font-semibold">{product.name}</h1>
            <div className="mt-2 flex items-center gap-2 text-sm text-muted-foreground">
              <RatingBadge rating={product.rating} />
              <span>· {formatCompactCount(product.soldCount)} terjual</span>
            </div>
          </header>

          <p className="text-3xl font-bold">
            {hasChoices ? formatRupiah(variant.price) : formatPriceRange(product.lowestPrice, product.highestPrice)}
          </p>

          {hasChoices && (
            <fieldset>
              <legend className="mb-2 text-sm font-medium">Pilih varian</legend>
              <div className="flex flex-wrap gap-2">
                {product.variants.map((v) => (
                  <label
                    key={v.id}
                    className={cn(
                      'cursor-pointer rounded-lg border px-3 py-1.5 text-sm transition-colors has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/50',
                      v.id === variantId
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'bg-card text-foreground/80 hover:border-foreground/30',
                      v.stock === 0 && 'opacity-50',
                    )}
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

          <p className={`text-sm ${available ? 'text-muted-foreground' : 'font-medium text-destructive'}`}>
            {available ? `Stok tersedia: ${variant.stock}` : 'Stok habis'}
          </p>

          <div className="flex items-center gap-3">
            <span className="text-sm font-medium" id="qty-label">Jumlah</span>
            <div role="group" aria-labelledby="qty-label" className="inline-flex items-center rounded-lg border bg-card">
              <Button
                variant="ghost"
                size="icon-lg"
                aria-label="Kurangi jumlah"
                disabled={!available || quantity <= 1}
                onClick={() => setQuantity((q) => q - 1)}
              >
                <Minus aria-hidden="true" />
              </Button>
              <output aria-live="polite" className="min-w-10 text-center text-sm">{quantity}</output>
              <Button
                variant="ghost"
                size="icon-lg"
                aria-label="Tambah jumlah"
                disabled={!available || quantity >= variant.stock}
                onClick={() => setQuantity((q) => q + 1)}
              >
                <Plus aria-hidden="true" />
              </Button>
            </div>
            {available && (
              <span className="text-sm text-muted-foreground">Subtotal {formatRupiah(Number(variant.price) * quantity)}</span>
            )}
          </div>

          <div className="flex flex-wrap gap-3">
            <Button size="lg" onClick={addToCart} disabled={!available || adding} className="h-10 px-5 font-semibold">
              {adding ? 'Menambahkan…' : 'Tambah ke keranjang'}
            </Button>
            <Button variant="outline" size="lg" disabled className="h-10 border-primary px-5 font-semibold text-primary">
              Beli sekarang
            </Button>
          </div>
          {cartMessage && (
            <Alert
              role={cartMessage.kind === 'error' ? 'alert' : 'status'}
              variant={cartMessage.kind === 'error' ? 'destructive' : 'default'}
              className={cartMessage.kind === 'error' ? 'border-destructive/30 bg-destructive/5' : 'border-primary/30 bg-primary/5'}
            >
              <AlertDescription className={cartMessage.kind === 'ok' ? 'text-foreground' : undefined}>
                <p>
                  {cartMessage.text}{' '}
                  {cartMessage.kind === 'ok' && (
                    <Link to="/cart" className="font-medium text-primary underline">
                      Lihat keranjang
                    </Link>
                  )}
                </p>
              </AlertDescription>
            </Alert>
          )}
          <Alert role="note" className="border-transparent bg-muted">
            <Info aria-hidden="true" />
            <AlertDescription>
              Checkout ("Beli sekarang") tersedia di tahap berikutnya. Menambahkan ke keranjang tidak memesan stok.
            </AlertDescription>
          </Alert>

          <Link
            to={`/stores/${product.store.slug}`}
            className="flex items-center gap-3 rounded-xl bg-card p-4 ring-1 ring-foreground/10 transition hover:ring-primary/50 outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            <span className="flex size-11 items-center justify-center rounded-full bg-primary/15 text-primary">
              <Store className="size-5" aria-hidden="true" />
            </span>
            <span>
              <span className="block font-medium">{product.store.name}</span>
              <span className="flex items-center gap-1 text-sm text-muted-foreground">
                <MapPin className="size-3.5" aria-hidden="true" /> {product.store.location}
              </span>
            </span>
          </Link>
        </div>
      </div>

      <section className="mt-10" aria-labelledby="product-description">
        <h2 id="product-description" className="mb-2 text-lg font-semibold">
          Deskripsi produk
        </h2>
        <p className="max-w-3xl whitespace-pre-line text-foreground/85">{product.description}</p>
        <p className="mt-3 text-xs text-muted-foreground">SKU: {variant.sku}</p>
      </section>
    </article>
  )
}
