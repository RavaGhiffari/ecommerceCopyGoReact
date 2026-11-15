import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Heart } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useAsync } from '../../hooks/useAsync'
import { addToWishlist, listWishlistIds, removeFromWishlist } from '../../services/wishlist'
import { useAuth } from '../auth/authContext'
import { messageFor } from '../auth/serverErrors'

/**
 * Heart toggle on the product page. Signed-out visitors go to login first and
 * come back here. The server is idempotent, so a double click cannot create a
 * duplicate; `busy` only keeps the button from flickering.
 */
export function WishlistButton({ productId }: { productId: string }) {
  const { user } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  if (!user) {
    return (
      <HeartButton
        saved={false}
        onClick={() => navigate(`/login?next=${encodeURIComponent(location.pathname)}`)}
      />
    )
  }
  // key: a different user (re-login) loads their own wishlist.
  return <SignedInButton key={user.id} productId={productId} userId={user.id} />
}

function SignedInButton({ productId, userId }: { productId: string; userId: string }) {
  const navigate = useNavigate()
  const { state } = useAsync(() => listWishlistIds(), `wishlist-ids:${userId}`)
  // After a click the server's answer is known; until then use the loaded ids.
  const [override, setOverride] = useState<boolean | null>(null)
  const [busy, setBusy] = useState(false)

  const loaded = state.status === 'success' ? state.data.productIds.includes(productId) : false
  const saved = override ?? loaded

  const toggle = async () => {
    setBusy(true)
    try {
      if (saved) {
        await removeFromWishlist(productId)
        setOverride(false)
        toast.success('Dihapus dari favorit.')
      } else {
        await addToWishlist(productId)
        setOverride(true)
        // No <Link> inside a toast: the Toaster is rendered outside the
        // router (App.tsx), so a Link there crashes. Use the toast's action.
        toast.success('Disimpan ke favorit.', {
          action: { label: 'Lihat favorit', onClick: () => navigate('/wishlist') },
        })
      }
    } catch (err) {
      toast.error(messageFor(err))
    } finally {
      setBusy(false)
    }
  }

  return <HeartButton saved={saved} disabled={busy || state.status === 'loading'} onClick={() => void toggle()} />
}

function HeartButton({ saved, disabled, onClick }: { saved: boolean; disabled?: boolean; onClick: () => void }) {
  return (
    <Button
      variant="outline"
      size="icon"
      aria-pressed={saved}
      aria-label={saved ? 'Hapus dari favorit' : 'Simpan ke favorit'}
      title={saved ? 'Hapus dari favorit' : 'Simpan ke favorit'}
      disabled={disabled}
      onClick={onClick}
      className="shrink-0"
    >
      <Heart aria-hidden="true" className={cn(saved && 'fill-rose-500 text-rose-500')} />
    </Button>
  )
}
