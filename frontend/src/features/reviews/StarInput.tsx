import { useId, useState } from 'react'
import { Star } from 'lucide-react'
import { cn } from '@/lib/utils'

const STAR_LABELS =['Sangat buruk', 'Buruk', 'Cukup', 'Bagus', 'Sangat bagus'] as const

/**
 * Pick 1–5 stars. Built from native radio inputs (visually hidden), so the
 * browser gives it keyboard support for free: Tab into the group, arrow keys
 * change the rating. Hovering previews a value before clicking.
 */
export function StarInput({
  value,
  onChange,
  invalid,
  describedBy,
}: {
  value: number // 0 = nothing chosen yet
  onChange: (rating: number) => void
  invalid?: boolean
  describedBy?: string
}) {
  const name = useId()
  const [hover, setHover] = useState(0)
  const shown = hover || value

  return (
    <div>
      <div
        role="radiogroup"
        aria-label="Rating"
        aria-invalid={invalid || undefined}
        aria-describedby={describedBy}
        className="flex items-center gap-1"
        onMouseLeave={() => setHover(0)}
      >
        {[1, 2, 3, 4, 5].map((n) => (
          <label
            key={n}
            onMouseEnter={() => setHover(n)}
            // relative: the sr-only (absolute) radio stays inside its star, so
            // focusing it never scrolls the page somewhere else.
            className="relative cursor-pointer rounded-md p-0.5 has-focus-visible:ring-3 has-focus-visible:ring-ring/50"
          >
            <input
              type="radio"
              name={name}
              value={n}
              checked={value === n}
              onChange={() => onChange(n)}
              className="sr-only"
              aria-label={`${n} bintang, ${STAR_LABELS[n - 1]}`}
            />
            <Star
              aria-hidden="true"
              className={cn(
                'size-8 transition-colors',
                n <= shown ? 'fill-amber-400 text-amber-400' : 'fill-muted text-muted-foreground/40',
              )}
            />
          </label>
        ))}
        <span className="ml-2 min-w-24 text-sm text-muted-foreground" aria-hidden="true">
          {shown > 0 ? STAR_LABELS[shown - 1] : 'Pilih rating'}
        </span>
      </div>
    </div>
  )
}
