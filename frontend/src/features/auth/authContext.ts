import { createContext, useContext } from 'react'
import type { LoginInput, RegisterInput, UpdateProfileInput, User } from '../../types/auth'

export type AuthStatus = 'loading' | 'authenticated' | 'anonymous' | 'error'

export interface AuthContextValue {
  status: AuthStatus
  user: User | null
  login: (input: LoginInput) => Promise<User>
  register: (input: RegisterInput) => Promise<User>
  logout: () => Promise<void>
  updateProfile: (input: UpdateProfileInput) => Promise<User>
  refresh: () => void
}

export const AuthContext = createContext<AuthContextValue | null>(null)

/**
 * Who is logged in, as last reported by the server.
 *
 * This is for display and navigation only. The server decides on every
 * request what this user may do; hiding a button here protects nothing.
 */
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>')
  return ctx
}
