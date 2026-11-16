import { useState, type FormEvent, type ReactNode } from 'react'
import { Search } from 'lucide-react'
import { toast } from 'sonner'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import type { AccountStatus } from '../../types/admin'
import { messageFor } from '../auth/serverErrors'
import { ACCOUNT_STATUS_LABEL } from './labels'

export function AccountStatusBadge({ status }: { status: AccountStatus }) {
  return (
    <Badge
      className={cn(
        'rounded-md',
        status === 'active' ? 'bg-emerald-100 text-emerald-800' : 'bg-red-100 text-red-800',
      )}
    >
      {ACCOUNT_STATUS_LABEL[status]}
    </Badge>
  )
}

/** Row of filter chips ("Semua" + one per status); same look as the seller order filter. */
export function FilterChips<S extends string>({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: S | undefined
  options: { value: S; label: string }[]
  onChange: (value: S | '') => void
}) {
  const all: { value: S | ''; label: string }[] = [{ value: '', label: 'Semua' }, ...options]
  return (
    <div role="group" aria-label={label} className="flex flex-wrap gap-2">
      {all.map((f) => {
        const active = (value ?? '') === f.value
        return (
          <Button
            key={f.value || 'all'}
            size="sm"
            variant={active ? 'default' : 'outline'}
            aria-pressed={active}
            onClick={() => onChange(f.value)}
            className={cn('rounded-full', !active && 'text-muted-foreground')}
          >
            {f.label}
          </Button>
        )
      })}
    </div>
  )
}

/** Search box; the parent keys it on the URL's q so back/forward resets the text. */
export function AdminSearch({
  initial,
  placeholder,
  label,
  onSearch,
}: {
  initial: string
  placeholder: string
  label: string
  onSearch: (q: string) => void
}) {
  const [value, setValue] = useState(initial)
  const submit = (e: FormEvent) => {
    e.preventDefault()
    onSearch(value.trim())
  }
  return (
    <form role="search" onSubmit={submit} className="flex w-full max-w-sm">
      <Input
        type="search"
        aria-label={label}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder={placeholder}
        maxLength={100}
        className="h-9 min-w-0 flex-1 rounded-r-none border-r-0"
      />
      <Button type="submit" aria-label={`${label}: cari`} className="h-9 rounded-l-none px-3">
        <Search aria-hidden="true" />
      </Button>
    </form>
  )
}

/**
 * A button that runs an admin action, optionally behind an AlertDialog.
 * Errors become a toast with the Indonesian message; on success `onDone`
 * gets the server's answer and the caller shows its own toast.
 */
export function ActionButton<T>({
  label,
  run,
  onDone,
  confirm,
  destructive = false,
  disabled = false,
}: {
  label: string
  run: () => Promise<T>
  onDone: (result: T) => void
  confirm?: { title: string; description: ReactNode; action: string }
  destructive?: boolean
  disabled?: boolean
}) {
  const [busy, setBusy] = useState(false)

  const go = async () => {
    setBusy(true)
    try {
      onDone(await run())
    } catch (err) {
      toast.error(messageFor(err))
    } finally {
      setBusy(false)
    }
  }

  const button = (
    <Button
      size="sm"
      variant="outline"
      disabled={disabled || busy}
      onClick={confirm ? undefined : () => void go()}
      className={cn(destructive && 'text-destructive hover:text-destructive')}
    >
      {busy ? 'Menyimpan…' : label}
    </Button>
  )
  if (!confirm) return button

  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>{button}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{confirm.title}</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-2">{confirm.description}</div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Batal</AlertDialogCancel>
          <AlertDialogAction variant={destructive ? 'destructive' : 'default'} onClick={() => void go()}>
            {confirm.action}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

/** "12 hasil" line above a list. */
export function ResultCount({ total, noun }: { total: number; noun: string }) {
  return (
    <p className="text-sm text-muted-foreground" aria-live="polite">
      {total.toLocaleString('id-ID')} {noun}
    </p>
  )
}
