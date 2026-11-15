import { Star } from 'lucide-react'
import { cn } from '@/lib/utils'

/**
 * Read-only row of 5 stars. `value` may be an average (4.3): it is rounded to
 * whole stars for the picture, while screen readers hear the exact value.
 */
export function Stars({ value, className }: { value: number; className?: string }) {
  const filled = Math.round(value)
  return (
    <span
      role="img"
      aria-label={`${value.toFixed(1).replace('.', ',')} dari 5 bintang`}
      className={cn('inline-flex items-center gap-0.5', className)}
    >
      {[1, 2, 3, 4, 5].map((n) => (
        <Star
          key={n}
          aria-hidden="true"
          className={cn('size-4', n <= filled ? 'fill-amber-400 text-amber-400' : 'fill-muted text-muted-foreground/40')}
        />
      ))}
    </span>
  )
}
