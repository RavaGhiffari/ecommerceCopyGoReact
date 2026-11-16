import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Pencil, Plus } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { TableSkeleton } from '../components/shared/Skeletons'
import { EmptyState, ErrorState } from '../components/shared/StateViews'
import { CategoryFormDialog } from '../features/admin/CategoryFormDialog'
import { useAsync } from '../hooks/useAsync'
import { usePageTitle } from '../hooks/usePageTitle'
import { listCategories } from '../services/catalog'
import type { Category } from '../types/marketplace'

const HEADERS = ['Kategori', 'Slug', 'Urutan', '']

/**
 * /admin/categories: the public category list, plus create and edit through
 * the admin API. There is no delete: products point at categories.
 */
export function AdminCategoriesPage() {
  usePageTitle('Kategori · Admin')
  const { state, reload } = useAsync(() => listCategories(), 'admin-categories')
  // undefined = closed; null = creating; a category = editing it.
  const [editing, setEditing] = useState<Category | null | undefined>(undefined)

  const onSaved = (saved: Category, created: boolean) => {
    setEditing(undefined)
    toast.success(created ? `Kategori “${saved.name}” ditambahkan.` : `Kategori “${saved.name}” disimpan.`)
    reload()
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Kategori</h1>
          <p className="text-sm text-muted-foreground">Menu kategori di header ikut berubah setelah halaman dimuat ulang.</p>
        </div>
        <Button onClick={() => setEditing(null)} disabled={state.status !== 'success'}>
          <Plus aria-hidden="true" /> Tambah kategori
        </Button>
      </div>

      {state.status === 'loading' && <TableSkeleton label="Memuat kategori" headers={HEADERS} rows={8} />}
      {state.status === 'error' && <ErrorState error={state.error} onRetry={reload} title="Gagal memuat kategori" />}
      {state.status === 'success' && state.data.length === 0 && (
        <EmptyState title="Belum ada kategori" description="Tambahkan kategori utama dulu, lalu subkategorinya." />
      )}
      {state.status === 'success' && state.data.length > 0 && (
        <Card className="py-0">
          <Table>
            <TableHeader className="bg-muted/50">
              <TableRow>
                <TableHead scope="col" className="px-4">Kategori</TableHead>
                <TableHead scope="col">Slug</TableHead>
                <TableHead scope="col" className="text-right">Urutan</TableHead>
                <TableHead scope="col" className="px-4"><span className="sr-only">Aksi</span></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tree(state.data).map(({ category: c, depth }) => (
                <TableRow key={c.id} className={depth === 0 ? 'bg-muted/20' : undefined}>
                  <TableCell className={depth === 0 ? 'px-4 font-medium' : 'pl-10'}>
                    {depth > 0 && (
                      <span aria-hidden="true" className="mr-1 text-muted-foreground">
                        └
                      </span>
                    )}
                    <Link to={`/categories/${c.slug}`} className="hover:text-primary hover:underline">
                      {c.name}
                    </Link>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{c.slug}</TableCell>
                  <TableCell className="text-right tabular-nums">{c.sortOrder}</TableCell>
                  <TableCell className="px-4 text-right">
                    <Button variant="outline" size="sm" onClick={() => setEditing(c)} aria-label={`Ubah ${c.name}`}>
                      <Pencil aria-hidden="true" /> Ubah
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      {state.status === 'success' && (
        <CategoryFormDialog
          open={editing !== undefined}
          onOpenChange={(open) => {
            if (!open) setEditing(undefined)
          }}
          category={editing ?? undefined}
          all={state.data}
          onSaved={onSaved}
        />
      )}
    </div>
  )
}

/** Top-level categories, each followed by its subcategories, both in sortOrder then name order. */
function tree(all: Category[]): { category: Category; depth: number }[] {
  const order = (a: Category, b: Category) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name, 'id')
  const out: { category: Category; depth: number }[] = []
  for (const parent of all.filter((c) => c.parentId === null).sort(order)) {
    out.push({ category: parent, depth: 0 })
    for (const child of all.filter((c) => c.parentId === parent.id).sort(order)) {
      out.push({ category: child, depth: 1 })
    }
  }
  return out
}
