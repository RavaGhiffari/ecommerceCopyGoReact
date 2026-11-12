import type { ProductImage as ProductImageData } from '../../types/marketplace'

interface ProductImageProps {
  id: string
  name: string
  image?: ProductImageData
  className?: string
}

/**
 * Shows the product photo, or a generated placeholder when there is none.
 * The hue is derived from the id so each product keeps the same colour.
 */
export function ProductImage({ id, name, image, className = '' }: ProductImageProps) {
  if (image) {
    return (
      <img
        src={image.url}
        alt={image.alt}
        loading="lazy"
        className={`aspect-square w-full object-cover ${className}`}
      />
    )
  }

  let hash = 0
  for (const ch of id) hash = (hash * 31 + ch.charCodeAt(0)) % 360
  const initials = name
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0])
    .join('')
    .toUpperCase()

  return (
    <div
      role="img"
      aria-label={`Foto ${name} belum tersedia`}
      style={{ backgroundImage: `linear-gradient(135deg, hsl(${hash} 70% 85%), hsl(${(hash + 40) % 360} 65% 72%))` }}
      className={`flex aspect-square w-full items-center justify-center ${className}`}
    >
      <span className="text-3xl font-bold tracking-wide text-white/90 drop-shadow-sm">{initials}</span>
    </div>
  )
}
