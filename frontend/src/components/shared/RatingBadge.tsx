import { Star } from 'lucide-react'
import { formatCompactCount } from '../../lib/format'
import type { Rating } from '../../types/marketplace'

/** Inline rating text (star + average + count); not a shadcn Badge on purpose, it sits inside card text. */
export function RatingBadge({ rating }: { rating: Rating }) {
  if (rating.count === 0) {
    return <span className="text-xs text-muted-foreground">Belum ada ulasan</span>
  }
  return (
    <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
      <Star className="size-3.5 fill-amber-400 text-amber-400" aria-hidden="true" />
      <span className="font-medium text-foreground">{rating.average.toFixed(1).replace('.', ',')}</span>
      <span>({formatCompactCount(rating.count)})</span>
    </span>
  )
}
