/**
 * The page to return to after login, from ?next=. Only same-site paths are
 * accepted: "/account" is fine, "https://evil.example" and "//evil.example"
 * are not (that would be an open redirect).
 */
export function safeNextPath(next: string | null): string {
  if (!next || !next.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) return '/'
  return next
}
