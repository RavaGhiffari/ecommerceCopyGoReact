import { useId } from 'react'
import type { ProductSort } from '../../types/marketplace'
import { SORT_OPTIONS } from './query'

export function SortSelect({
  value,
  onChange,
}: {
  value: ProductSort
  onChange: (sort: ProductSort) => void
}) {
  const id = useId()
  return (
    <div className="flex items-center gap-2 text-sm">
      <label htmlFor={id} className="text-slate-600">
        Urutkan
      </label>
      <select
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value as ProductSort)}
        className="rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm text-slate-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
      >
        {SORT_OPTIONS.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  )
}
