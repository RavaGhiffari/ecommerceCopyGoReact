import type { AccountStatus, AuditLog } from '../../types/admin'
import type { Role } from '../../types/auth'
import type { ReviewStatus } from '../../types/reviews'

export const ACCOUNT_STATUSES: readonly AccountStatus[] = ['active', 'suspended']
export const REVIEW_STATUSES: readonly ReviewStatus[] = ['visible', 'hidden']

export const ACCOUNT_STATUS_LABEL: Record<AccountStatus, string> = {
  active: 'Aktif',
  suspended: 'Ditangguhkan',
}

export const REVIEW_STATUS_LABEL: Record<ReviewStatus, string> = {
  visible: 'Tampil',
  hidden: 'Disembunyikan',
}

export const ROLE_LABEL: Record<Role, string> = {
  customer: 'Pembeli',
  seller: 'Penjual',
  admin: 'Admin',
}

export const AUDIT_ACTION_LABEL: Record<string, string> = {
  'user.suspend': 'Menangguhkan pengguna',
  'user.reactivate': 'Mengaktifkan pengguna',
  'store.suspend': 'Menangguhkan toko',
  'store.reactivate': 'Mengaktifkan toko',
  'category.create': 'Membuat kategori',
  'category.update': 'Mengubah kategori',
  'review.hide': 'Menyembunyikan ulasan',
  'review.unhide': 'Menampilkan ulasan',
}

const str = (v: unknown) => (typeof v === 'string' || typeof v === 'number' ? String(v) : '')
const field = (v: unknown, key: string) =>
  typeof v === 'object' && v !== null ? str((v as Record<string, unknown>)[key]) : ''

/**
 * One line describing the target of an audit row, built from the details the
 * backend stored with it (see the audit.Write calls in the 6d services).
 * Unknown actions fall back to the target type and id.
 */
export function auditSummary(log: AuditLog): string {
  const d = log.details
  switch (log.action) {
    case 'user.suspend': {
      const n = Number(d.sessionsEnded ?? 0)
      return `${str(d.email)}${n > 0 ? ` · ${n} sesi diakhiri` : ''}`
    }
    case 'user.reactivate':
      return str(d.email)
    case 'store.suspend':
    case 'store.reactivate':
      return `${str(d.name)} (${str(d.slug)})`
    case 'category.create':
      return `${str(d.name)} (${str(d.slug)})`
    case 'category.update': {
      const before = field(d.before, 'name')
      const after = field(d.after, 'name')
      const slugBefore = field(d.before, 'slug')
      const slugAfter = field(d.after, 'slug')
      const name = before === after ? after : `${before} → ${after}`
      return slugBefore === slugAfter ? name : `${name} (${slugBefore} → ${slugAfter})`
    }
    case 'review.hide':
    case 'review.unhide':
      return `Ulasan bintang ${str(d.rating)}`
    default:
      return `${log.targetType} ${log.targetId}`
  }
}
