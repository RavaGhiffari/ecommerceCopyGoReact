import { useState } from 'react'
import { MapPin, Plus } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { cn } from '@/lib/utils'
import { MAX_ADDRESSES, type Address } from '../../types/address'
import { AddressFormDialog } from '../addresses/AddressFormDialog'

interface Props {
  addresses: Address[]
  value: string | null
  onChange: (id: string) => void
  /** Called after a new address was saved; the caller reloads and selects it. */
  onAdded: (address: Address) => void
  disabled?: boolean
}

/** Address choice for /checkout, with "add new" in the same Dialog as /account. */
export function AddressPicker({ addresses, value, onChange, onAdded, disabled }: Props) {
  const [adding, setAdding] = useState(false)
  const full = addresses.length >= MAX_ADDRESSES

  return (
    <Card className="block p-5">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h2 id="address-heading" className="flex items-center gap-2 text-lg font-semibold">
          <MapPin className="size-5 text-primary" aria-hidden="true" /> Alamat pengiriman
        </h2>
        <Button variant="outline" size="sm" onClick={() => setAdding(true)} disabled={disabled || full}>
          <Plus aria-hidden="true" /> Alamat baru
        </Button>
      </div>

      {addresses.length === 0 ? (
        <div className="rounded-lg border border-dashed px-4 py-6 text-center">
          <p className="font-medium">Belum ada alamat</p>
          <p className="text-sm text-muted-foreground">Tambahkan alamat pengiriman untuk melanjutkan.</p>
        </div>
      ) : (
        <RadioGroup
          aria-labelledby="address-heading"
          value={value ?? ''}
          onValueChange={onChange}
          disabled={disabled}
          className="gap-2"
        >
          {addresses.map((a) => (
            <label
              key={a.id}
              htmlFor={`address-${a.id}`}
              className={cn(
                'flex cursor-pointer gap-3 rounded-lg border p-3 transition-colors hover:bg-muted/50',
                value === a.id && 'border-primary bg-primary/5',
              )}
            >
              <RadioGroupItem id={`address-${a.id}`} value={a.id} className="mt-1" aria-label={`Alamat ${a.label}`} />
              <span className="min-w-0 text-sm">
                <span className="flex flex-wrap items-center gap-2 font-medium">
                  {a.label} {a.isDefault && <Badge>Utama</Badge>}
                </span>
                <span className="block">
                  {a.recipientName} · <span className="text-muted-foreground">{a.phone}</span>
                </span>
                <span className="block text-muted-foreground">
                  {a.addressLine}, {a.city}, {a.province} {a.postalCode}
                </span>
              </span>
            </label>
          ))}
        </RadioGroup>
      )}

      <AddressFormDialog
        open={adding}
        onOpenChange={setAdding}
        isFirst={addresses.length === 0}
        onSaved={(saved) => {
          setAdding(false)
          onAdded(saved)
        }}
      />
    </Card>
  )
}
