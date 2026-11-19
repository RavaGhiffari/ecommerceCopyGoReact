import { LineSkeleton, LoadingRegion } from '../components/shared/Skeletons'

// Shown only when the very first page is a lazy one (e.g. a bookmarked /cart):
// the router has nothing to render until that chunk has arrived.
export function FirstLoad() {
  return (
    <LoadingRegion label="Memuat halaman" className="mx-auto max-w-5xl space-y-3 p-6">
      <LineSkeleton className="h-8 w-1/3" />
      <LineSkeleton className="w-full" />
      <LineSkeleton className="w-2/3" />
    </LoadingRegion>
  )
}
