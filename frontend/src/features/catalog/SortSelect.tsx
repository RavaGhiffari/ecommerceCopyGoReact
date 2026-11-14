import { useId } from 'react'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
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
      <Label htmlFor={id} className="font-normal text-muted-foreground">
        Urutkan
      </Label>
      {/* onValueChange gives a string; it is always one of SORT_OPTIONS' values. */}
      <Select value={value} onValueChange={(v) => onChange(v as ProductSort)}>
        <SelectTrigger id={id} className="min-w-40 bg-card">
          <SelectValue />
        </SelectTrigger>
        <SelectContent position="popper" align="end">
          {SORT_OPTIONS.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
