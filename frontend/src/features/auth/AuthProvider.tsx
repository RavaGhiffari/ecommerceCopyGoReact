import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import * as authService from '../../services/auth'
import type { LoginInput, RegisterInput, UpdateProfileInput, User } from '../../types/auth'
import { AuthContext, type AuthContextValue, type AuthStatus } from './authContext'

/**
 * Loads the current user once (GET /api/v1/me) and keeps it in React state.
 * Login / register / logout update that state from the server's answer.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<{ status: AuthStatus; user: User | null }>({
    status: 'loading',
    user: null,
  })
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    const controller = new AbortController()
    authService
      .getMe(controller.signal)
      .then((user) => setState({ status: user ? 'authenticated' : 'anonymous', user }))
      .catch(() => {
        // Server unreachable: treat as logged out for display, but remember
        // it was an error so the UI can say so.
        if (!controller.signal.aborted) setState({ status: 'error', user: null })
      })
    return () => controller.abort()
  }, [attempt])

  const setUser = useCallback((user: User | null) => {
    setState({ status: user ? 'authenticated' : 'anonymous', user })
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      ...state,
      login: async (input: LoginInput) => {
        const user = await authService.login(input)
        setUser(user)
        return user
      },
      register: async (input: RegisterInput) => {
        const user = await authService.register(input)
        setUser(user)
        return user
      },
      logout: async () => {
        await authService.logout()
        setUser(null)
      },
      updateProfile: async (input: UpdateProfileInput) => {
        const user = await authService.updateProfile(input)
        setUser(user)
        return user
      },
      refresh: () => setAttempt((n) => n + 1),
    }),
    [state, setUser],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
