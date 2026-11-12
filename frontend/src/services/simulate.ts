// Prototype-only helpers so loading / empty / error states can be checked by
// hand. Add ?simulate=slow, ?simulate=empty or ?simulate=error to any page URL.
// Deleted together with src/mocks when the real API replaces these services.
import { ApiError } from '../lib/api'

export type Simulation = 'slow' | 'empty' | 'error' | null

export function currentSimulation(): Simulation {
  const value = new URLSearchParams(window.location.search).get('simulate')
  return value === 'slow' || value === 'empty' || value === 'error' ? value : null
}

/** Pretends to be a network call: waits, and fails when ?simulate=error. */
export async function mockRequest(): Promise<Simulation> {
  const sim = currentSimulation()
  await new Promise((resolve) => setTimeout(resolve, sim === 'slow' ? 2500 : 300))
  if (sim === 'error') {
    throw new ApiError(500, 'SIMULATED_ERROR', 'Kesalahan simulasi dari server contoh.')
  }
  return sim
}
