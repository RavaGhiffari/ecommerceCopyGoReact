import { Star } from 'lucide-react'
import { formatCompactCount } from '../../lib/format'
import type { Rating } from '../../types/marketplace'

export function RatingBadge({ rating }: { rating: Rating }) {
  if (rating.count === 0) {
    return <span className="text-xs text-slate-500">Belum ada ulasan</span>
  }
  return (
    <span className="inline-flex items-center gap-1 text-xs text-slate-600">
      <Star className="size-3.5 fill-amber-400 text-amber-400" aria-hidden="true" />
      <span className="font-medium text-slate-800">{rating.average.toFixed(1).replace('.', ',')}</span>
      <span>({formatCompactCount(rating.count)})</span>
    </span>
  )
}
