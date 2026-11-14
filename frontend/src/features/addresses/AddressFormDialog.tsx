import { useState } from 'react'
import { useForm } from 'react-hook-form'
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
import { FormAlert, TextField } from '../../components/shared/TextField'
import { applyServerError } from '../auth/serverErrors'
import { createAddress, updateAddress } from '../../services/addresses'
import type { Address } from '../../types/address'
import { ADDRESS_FIELDS, addressSchema, type AddressValues } from './schema'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** The address to edit; undefined = create a new one. */
  address?: Address
  /** True when the user has no address yet (the new one becomes the default anyway). */
  isFirst: boolean
  onSaved: (saved: Address) => void
}

/**
 * Create / edit dialog. Radix Dialog handles focus trapping, Escape, and
 * returning focus to the button that opened it.
 */
export function AddressFormDialog({ open, onOpenChange, address, isFirst, onSaved }: Props) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{address ? 'Ubah alamat' : 'Tambah alamat'}</DialogTitle>
          <DialogDescription>Alamat ini disalin ke pesanan saat checkout.</DialogDescription>
        </DialogHeader>
        {/* Mounted only while open (Radix unmounts closed content), and keyed,
            so every opening starts from the address's current values. */}
        <AddressForm key={address?.id ?? 'new'} address={address} isFirst={isFirst} onSaved={onSaved} />
      </DialogContent>
    </Dialog>
  )
}

function AddressForm({ address, isFirst, onSaved }: Pick<Props, 'address' | 'isFirst' | 'onSaved'>) {
  const [formError, setFormError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<AddressValues>({
    resolver: zodResolver(addressSchema),
    defaultValues: {
      label: address?.label ?? '',
      recipientName: address?.recipientName ?? '',
      phone: address?.phone ?? '',
      addressLine: address?.addressLine ?? '',
      city: address?.city ?? '',
      province: address?.province ?? '',
      postalCode: address?.postalCode ?? '',
      isDefault: false,
    },
  })

  const onSubmit = async ({ isDefault, ...fields }: AddressValues) => {
    setFormError(null)
    try {
      const saved = address ? await updateAddress(address.id, fields) : await createAddress({ ...fields, isDefault })
      onSaved(saved)
    } catch (err) {
      setFormError(applyServerError(err, setError, ADDRESS_FIELDS))
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
      <FormAlert message={formError} />
      <div className="grid gap-4 sm:grid-cols-2">
        <TextField label="Label" placeholder="Rumah, Kantor…" registration={register('label')} error={errors.label?.message} />
        <TextField
          label="Nama penerima"
          autoComplete="name"
          registration={register('recipientName')}
          error={errors.recipientName?.message}
        />
      </div>
      <TextField
        label="Nomor telepon"
        type="tel"
        autoComplete="tel"
        hint="Contoh: 0812-3456-7890"
        registration={register('phone')}
        error={errors.phone?.message}
      />
      <TextField
        label="Alamat lengkap"
        autoComplete="street-address"
        placeholder="Jalan, nomor rumah, RT/RW, kelurahan, kecamatan"
        registration={register('addressLine')}
        error={errors.addressLine?.message}
      />
      <div className="grid gap-4 sm:grid-cols-3">
        <TextField label="Kota/kabupaten" autoComplete="address-level2" registration={register('city')} error={errors.city?.message} />
        <TextField label="Provinsi" autoComplete="address-level1" registration={register('province')} error={errors.province?.message} />
        <TextField
          label="Kode pos"
          inputMode="numeric"
          autoComplete="postal-code"
          maxLength={5}
          registration={register('postalCode')}
          error={errors.postalCode?.message}
        />
      </div>

      {!address &&
        (isFirst ? (
          <p className="text-sm text-muted-foreground">Alamat pertama otomatis menjadi alamat utama.</p>
        ) : (
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" className="size-4 accent-primary" {...register('isDefault')} />
            Jadikan alamat utama
          </label>
        ))}

      <DialogFooter>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Menyimpan…' : 'Simpan alamat'}
        </Button>
      </DialogFooter>
    </form>
  )
}
