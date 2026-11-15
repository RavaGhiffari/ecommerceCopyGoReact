import { useId, useState } from 'react'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { FieldMessage } from '../../components/shared/FieldMessage'
import { TextAreaField } from '../../components/shared/TextAreaField'
import { FormAlert } from '../../components/shared/TextField'
import { applyServerError } from '../auth/serverErrors'
import { createReview, updateReview } from '../../services/reviews'
import type { Review } from '../../types/reviews'
import { charCount, MAX_COMMENT, reviewSchema, type ReviewValues } from './schema'
import { StarInput } from './StarInput'

export interface ReviewTarget {
  orderItemId: string
  productName: string
  variantName: string
}

/** An existing review to edit; absent = write a new one. */
export interface ExistingReview {
  id: string
  rating: number
  comment: string
}

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  target: ReviewTarget
  existing?: ExistingReview
  onSaved: (review: Review) => void
}

/** Write or edit the review of one purchased item (one review per item). */
export function ReviewDialog({ open, onOpenChange, target, existing, onSaved }: Props) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{existing ? 'Ubah ulasan' : 'Beri ulasan'}</DialogTitle>
          <DialogDescription>
            {target.productName} · {target.variantName}
          </DialogDescription>
        </DialogHeader>
        {/* Radix unmounts closed content; the key restarts the form per item. */}
        <ReviewForm key={existing?.id ?? target.orderItemId} target={target} existing={existing} onSaved={onSaved} />
      </DialogContent>
    </Dialog>
  )
}

function ReviewForm({ target, existing, onSaved }: Pick<Props, 'target' | 'existing' | 'onSaved'>) {
  const ratingMsgId = useId()
  const [formError, setFormError] = useState<string | null>(null)
  const {
    control,
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ReviewValues>({
    resolver: zodResolver(reviewSchema),
    defaultValues: { rating: existing?.rating ?? 0, comment: existing?.comment ?? '' },
  })
  const length = charCount(useWatch({ control, name: 'comment' }) ?? '')

  const onSubmit = async (values: ReviewValues) => {
    setFormError(null)
    try {
      const saved = existing
        ? await updateReview(existing.id, values)
        : await createReview({ orderItemId: target.orderItemId, ...values })
      onSaved(saved)
    } catch (err) {
      setFormError(applyServerError(err, setError, ['rating', 'comment']))
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
      <FormAlert message={formError} />
      <div className="space-y-1.5">
        <p className="text-sm font-medium">Rating</p>
        <Controller
          control={control}
          name="rating"
          render={({ field }) => (
            <StarInput
              value={field.value}
              onChange={field.onChange}
              invalid={!!errors.rating}
              describedBy={errors.rating ? `${ratingMsgId}-error` : undefined}
            />
          )}
        />
        <FieldMessage id={ratingMsgId} error={errors.rating?.message} />
      </div>
      <TextAreaField
        label="Ulasan (opsional)"
        placeholder="Ceritakan kualitas barang, kesesuaian dengan deskripsi, dan pengirimannya."
        rows={5}
        registration={register('comment')}
        error={errors.comment?.message}
        hint={`${length}/${MAX_COMMENT} karakter`}
      />
      <DialogFooter>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Menyimpan…' : existing ? 'Simpan perubahan' : 'Kirim ulasan'}
        </Button>
      </DialogFooter>
    </form>
  )
}
