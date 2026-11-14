/**
 * The line under a form control: the error if there is one, otherwise the
 * hint. Ids match the control's aria-describedby (`${id}-error` / `${id}-hint`).
 */
export function FieldMessage({ id, error, hint }: { id: string; error?: string; hint?: string }) {
  if (error) {
    return (
      <p id={`${id}-error`} className="text-sm text-destructive">
        {error}
      </p>
    )
  }
  if (hint) {
    return (
      <p id={`${id}-hint`} className="text-xs text-muted-foreground">
        {hint}
      </p>
    )
  }
  return null
}
