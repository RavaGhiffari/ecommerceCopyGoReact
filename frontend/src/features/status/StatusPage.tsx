import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
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
    <main className="min-h-screen bg-muted/50 px-4 py-12">
      <div className="mx-auto max-w-xl">
        <h1 className="text-2xl font-semibold">System status</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Live checks against the Go API. Stop PostgreSQL with{' '}
          <code className="rounded bg-muted px-1">docker compose stop postgres</code> and
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

        <Button size="lg" onClick={reloadAll} className="mt-6">
          Refresh
        </Button>
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
    <Card className="block p-4 shadow-sm">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="font-medium">{title}</h2>
          <p className="text-sm text-muted-foreground">{description}</p>
          <p className="mt-1 font-mono text-xs text-muted-foreground">{endpoint}</p>
        </div>
        <StatusBadge state={state} />
      </div>

      {state.kind === 'error' && (
        <p className="mt-3 rounded-lg bg-destructive/5 px-3 py-2 text-sm text-destructive" role="alert">
          {state.error.message}{' '}
          <span className="font-mono text-xs">
            ({state.error.code}
            {state.error.status ? `, HTTP ${state.error.status}` : ''})
          </span>
        </p>
      )}
    </Card>
  )
}

function StatusBadge<T>({ state }: { state: CheckState<T> }) {
  if (state.kind === 'loading') {
    return <span className="text-sm text-muted-foreground">Checking…</span>
  }
  if (state.kind === 'ok') {
    return (
      <Badge className="bg-primary/15 text-primary">Up · {state.ms} ms</Badge>
    )
  }
  return (
    <Badge variant="destructive">Down</Badge>
  )
}
