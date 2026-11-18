import { describe, expect, it } from 'vitest'
import { plain } from '../test/fixtures'
import { formatCompactCount, formatCountdown, formatPriceRange, formatRupiah } from './format'

describe('formatRupiah', () => {
  it('formats the API decimal strings without cents', () => {
    expect(plain(formatRupiah('349000.00'))).toBe('Rp 349.000')
    expect(plain(formatRupiah(0))).toBe('Rp 0')
  })

  it('rounds cents for display only', () => {
    expect(plain(formatRupiah('100000.50'))).toBe('Rp 100.001')
  })
})

describe('formatPriceRange', () => {
  it('shows one price when all variants cost the same', () => {
    expect(plain(formatPriceRange('59000.00', '59000'))).toBe('Rp 59.000')
  })

  it('shows a range otherwise', () => {
    expect(plain(formatPriceRange('49000.00', '59000.00'))).toBe('Rp 49.000 - Rp 59.000')
  })
})

describe('formatCompactCount', () => {
  it.each([
    [0, '0'],
    [950, '950'],
    [1000, '1 rb'],
    [1250, '1,3 rb'],
    [21000, '21 rb'],
    [1_500_000, '1,5 jt'],
  ])('%d -> %s', (n, want) => {
    expect(formatCompactCount(n)).toBe(want)
  })
})

describe('formatCountdown', () => {
  it.each([
    [-5000, '0:00'],
    [0, '0:00'],
    [999, '0:00'],
    [61_000, '1:01'],
    [29 * 60_000 + 58_000, '29:58'],
    [3_723_000, '1:02:03'],
  ])('%d ms -> %s', (ms, want) => {
    expect(formatCountdown(ms)).toBe(want)
  })
})
