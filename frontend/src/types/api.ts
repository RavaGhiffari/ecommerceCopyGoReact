// TypeScript mirror of the backend envelope in
// backend/internal/platform/response/response.go. Keep the two in sync.

export interface ApiSuccess<T> {
  success: true
  data: T
  message?: string
}

export interface ApiErrorInfo {
  code: string
  message: string
  details?: unknown
}

export interface ApiFailure {
  success: false
  error: ApiErrorInfo
}

export type ApiEnvelope<T> = ApiSuccess<T> | ApiFailure

// GET /health  (backend/internal/modules/health/handler.go: Live)
export interface HealthData {
  status: 'ok'
}

// GET /ready   (backend/internal/modules/health/handler.go: Ready)
export interface ReadyData {
  status: 'ready'
  database: 'up'
}
