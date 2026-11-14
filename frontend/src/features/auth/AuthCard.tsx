import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'

export function AuthCard({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Card className="mx-auto max-w-md gap-6 py-6 shadow-sm sm:py-8">
      <CardHeader className="sm:px-8">
        {/* A real <h1>, not CardTitle (a <div>): pages are found by their heading. */}
        <h1 className="text-2xl font-semibold">{title}</h1>
      </CardHeader>
      <CardContent className="sm:px-8">{children}</CardContent>
    </Card>
  )
}

export function SubmitButton({ busy, children }: { busy: boolean; children: ReactNode }) {
  return (
    <Button type="submit" size="lg" disabled={busy} className="h-10 w-full font-semibold">
      {busy ? 'Memproses…' : children}
    </Button>
  )
}
