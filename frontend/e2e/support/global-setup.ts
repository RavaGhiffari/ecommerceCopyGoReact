// Runs once before the suite: fail fast with a useful message when the E2E
// API is not running, instead of 60 seconds of timeouts per test.
import { E2E_API } from '../../playwright.config'

export default async function globalSetup() {
  const deadline = Date.now() + 30_000
  let last = ''
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`${E2E_API}/ready`)
      if (res.ok) return
      last = `HTTP ${res.status}`
    } catch (err) {
      last = String(err)
    }
    await new Promise((r) => setTimeout(r, 1000))
  }
  throw new Error(
    `The E2E API at ${E2E_API} is not ready (${last}). ` +
      'Start it with `make e2e-up` (after `make e2e-reset`), or run everything with `make e2e`.',
  )
}
