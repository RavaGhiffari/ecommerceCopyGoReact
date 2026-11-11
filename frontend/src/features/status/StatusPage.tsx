import type { HealthData, ReadyData } from '../../types/api'
import { useApiCheck, type CheckState } from './useApiCheck'

/**
 * Phase 0 developer page: proves browser -> Vite proxy -> Go API -> PostgreSQL
 * works end to end. Replaced by the marketplace homepage in Phase 3.
 */
export function StatusPage() {
  const health = useApiCheck<HealthData>('/health')
  const ready = useApiCheck<ReadyData>('/ready')

  const reloadAll = () => {
    health.reload()
    ready.reload()
  }

  return (
    <main className="min-h-screen bg-slate-50 px-4 py-12 text-slate-900">
      <div className="mx-auto max-w-xl">
        <h1 className="text-2xl font-semibold">System status</h1>
        <p className="mt-1 text-sm text-slate-600">
          Live checks against the Go API. Stop PostgreSQL with{' '}
          <code className="rounded bg-slate-200 px-1">docker compose stop postgres</code> and
          press Refresh to see the failure path.
        </p>

        <div className="mt-6 space-y-3">
          <CheckCard
            title="API"
            endpoint="GET /health"
            description="Is the Go process running?"
            state={health.state}
          />
          <CheckCard
            title="Database"
            endpoint="GET /ready"
            description="Can the API run a query on PostgreSQL?"
            state={ready.state}
          />
        </div>

        <button
          type="button"
          onClick={reloadAll}
          className="mt-6 rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600 focus-visible:ring-offset-2"
        >
          Refresh
        </button>
      </div>
    </main>
  )
}

interface CheckCardProps<T> {
  title: string
  endpoint: string
  description: string
  state: CheckState<T>
}

function CheckCard<T>({ title, endpoint, description, state }: CheckCardProps<T>) {
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="font-medium">{title}</h2>
          <p className="text-sm text-slate-600">{description}</p>
          <p className="mt-1 font-mono text-xs text-slate-500">{endpoint}</p>
        </div>
        <StatusBadge state={state} />
      </div>

      {state.kind === 'error' && (
        <p className="mt-3 rounded bg-red-50 px-3 py-2 text-sm text-red-800" role="alert">
          {state.error.message}{' '}
          <span className="font-mono text-xs">
            ({state.error.code}
            {state.error.status ? `, HTTP ${state.error.status}` : ''})
          </span>
        </p>
      )}
    </section>
  )
}

function StatusBadge<T>({ state }: { state: CheckState<T> }) {
  if (state.kind === 'loading') {
    return <span className="text-sm text-slate-500">Checking…</span>
  }
  if (state.kind === 'ok') {
    return (
      <span className="whitespace-nowrap rounded-full bg-emerald-100 px-2.5 py-0.5 text-sm font-medium text-emerald-800">
        Up · {state.ms} ms
      </span>
    )
  }
  return (
    <span className="whitespace-nowrap rounded-full bg-red-100 px-2.5 py-0.5 text-sm font-medium text-red-800">
      Down
    </span>
  )
}
