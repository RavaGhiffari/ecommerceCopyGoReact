import { describe, expect, it } from 'vitest'
import { DEFAULT_SORT, parsePage, parseSort } from './query'

describe('URL parameters are user input', () => {
  it.each(['popular', 'newest', 'price_asc', 'price_desc', 'rating'])('keeps the known sort %s', (s) => {
    expect(parseSort(s)).toBe(s)
  })

  it.each([null, '', 'cheapest', 'RATING', "rating'--"])('falls back to the default sort for %j', (s) => {
    expect(parseSort(s)).toBe(DEFAULT_SORT)
  })

  it.each([
    ['3', 3],
    [null, 1],
    ['0', 1],
    ['-2', 1],
    ['abc', 1],
  ])('page %j -> %d', (v, want) => {
    expect(parsePage(v)).toBe(want)
  })
})
