import { describe, expect, it } from 'vitest'
import { safeNextPath } from './redirect'

describe('safeNextPath (open-redirect guard for ?next=)', () => {
  it.each(['/account', '/cart?x=1', '/stores/a/products/b#ulasan'])('keeps the same-site path %s', (p) => {
    expect(safeNextPath(p)).toBe(p)
  })

  it.each([null, '', 'account', 'https://evil.example', '//evil.example', '/\\evil.example', 'javascript:alert(1)'])(
    'falls back to / for %s',
    (p) => {
      expect(safeNextPath(p)).toBe('/')
    },
  )
})
