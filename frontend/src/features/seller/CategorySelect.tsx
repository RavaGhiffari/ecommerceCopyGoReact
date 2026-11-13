import { useId } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'
import type { Category } from '../../types/marketplace'
import { useCategories } from '../catalog/hooks'

const selectClass = (error?: string) =>
  `w-full rounded-md border bg-white px-3 py-2 text-sm focus:outline-none focus-visible:ring-2 ${
    error ? 'border-red-400 focus-visible:ring-red-500' : 'border-slate-300 focus-visible:ring-emerald-600'
  }`

/**
 * Category picker for products. Products go on a LEAF category: subcategories
 * are listed under their parent (<optgroup>), and a parent with
 * subcategories cannot be picked itself. The API enforces the same rule.
 *
 * The registered <select> is only mounted once the categories have loaded.
 * react-hook-form writes the form's default value into the element when it
 * mounts; if the matching <option> did not exist yet, the browser would fall
 * back to "Pilih kategori" and the editor would show the wrong category.
 */
export function CategorySelect({ registration, error }: { registration: UseFormRegisterReturn; error?: string }) {
  const id = useId()
  const { state, reload } = useCategories()

  return (
    <div>
      <label htmlFor={id} className="mb-1 block text-sm font-medium text-slate-700">
        Kategori
      </label>
      {state.status === 'success' ? (
        <select
          id={id}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? `${id}-error` : undefined}
          className={selectClass(error)}
          {...registration}
        >
          <option value="">Pilih kategori</option>
          <CategoryOptions categories={state.data} />
        </select>
      ) : (
        <select id={id} disabled className={selectClass()}>
          <option>{state.status === 'loading' ? 'Memuat kategori…' : 'Kategori tidak tersedia'}</option>
        </select>
      )}
      {state.status === 'error' && (
        <p className="mt-1 text-sm text-red-600">
          Gagal memuat kategori.{' '}
          <button type="button" onClick={reload} className="underline">
            Coba lagi
          </button>
        </p>
      )}
      {error && (
        <p id={`${id}-error`} className="mt-1 text-sm text-red-600">
          {error}
        </p>
      )}
    </div>
  )
}

function CategoryOptions({ categories }: { categories: Category[] }) {
  return categories
    .filter((c) => c.parentId === null)
    .map((parent) => {
      const children = categories.filter((c) => c.parentId === parent.id)
      return children.length > 0 ? (
        <optgroup key={parent.id} label={parent.name}>
          {children.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </optgroup>
      ) : (
        <option key={parent.id} value={parent.id}>
          {parent.name}
        </option>
      )
    })
}
