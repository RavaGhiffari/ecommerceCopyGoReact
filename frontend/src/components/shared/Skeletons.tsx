import type { ReactNode } from 'react'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { cn } from '@/lib/utils'

// Loading placeholders shaped like the content they stand in for, so the page
// does not jump when the data arrives. Screen readers hear the label of the
// LoadingRegion once; the grey bars themselves are hidden from them.

/** Wraps a skeleton: one "loading" announcement, bars hidden from assistive tech. */
export function LoadingRegion({ label, className, children }: { label: string; className?: string; children: ReactNode }) {
  return (
    <div role="status" aria-label={label} className={className}>
      <div aria-hidden="true" className="contents">
        {children}
      </div>
    </div>
  )
}

/** A text line of a given width (Tailwind width class). */
export function LineSkeleton({ className }: { className?: string }) {
  return <Skeleton className={cn('h-4', className)} />
}

/** One list row: square thumbnail, two or three lines, and an optional right column. */
export function ItemRowSkeleton({ thumb = 'w-20', trailing = true }: { thumb?: string; trailing?: boolean }) {
  return (
    <div className="flex items-center gap-4 p-4">
      <Skeleton className={cn('aspect-square shrink-0 rounded-md', thumb)} />
      <div className="min-w-0 flex-1 space-y-2">
        <LineSkeleton className="w-3/4" />
        <LineSkeleton className="h-3 w-1/3" />
        <LineSkeleton className="w-1/4" />
      </div>
      {trailing && (
        <div className="flex flex-col items-end gap-3">
          <LineSkeleton className="w-20" />
          <Skeleton className="h-8 w-24" />
        </div>
      )}
    </div>
  )
}

// Varying widths so the rows do not look like a barcode.
const CELL_WIDTHS = ['w-3/4', 'w-1/2', 'w-2/3', 'w-1/3', 'w-5/6']

/** A table with the real column headers and placeholder cells. */
export function TableSkeleton({ headers, rows = 5, label }: { headers: string[]; rows?: number; label: string }) {
  return (
    <LoadingRegion label={label}>
      <Card className="py-0">
        <Table>
          <TableHeader className="bg-muted/50">
            <TableRow>
              {headers.map((h, i) => (
                <TableHead key={i} scope="col" className="px-4">
                  {h}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {Array.from({ length: rows }, (_, r) => (
              <TableRow key={r} className="hover:bg-transparent">
                {headers.map((_, c) => (
                  <TableCell key={c} className="px-4 py-3">
                    <LineSkeleton className={CELL_WIDTHS[(r + c) % CELL_WIDTHS.length]} />
                    {c === 0 && <LineSkeleton className="mt-1.5 h-3 w-1/2" />}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
    </LoadingRegion>
  )
}

/** A stat tile: label, big number, small caption. */
export function StatTileSkeleton() {
  return (
    <Card className="gap-2 p-4">
      <LineSkeleton className="h-3.5 w-1/2" />
      <Skeleton className="h-7 w-3/4" />
      <LineSkeleton className="h-3 w-2/3" />
    </Card>
  )
}
