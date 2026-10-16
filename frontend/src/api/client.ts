import createClient from 'openapi-fetch'

import type { components, paths } from '@/api/schema'

export type Expense = components['schemas']['Expense']
export type ExpenseInput = components['schemas']['ExpenseInput']
export type ExpenseList = components['schemas']['ExpenseList']
export type Summary = components['schemas']['Summary']
export type User = components['schemas']['User']
type ErrorBody = components['schemas']['Error']

/**
 * Typed client generated from backend/api/openapi.yaml (npm run gen:api).
 * Paths already start with /api: in production nginx serves the API on the
 * same origin, in development Vite proxies it.
 */
export const api = createClient<paths>({
  baseUrl: globalThis.location?.origin ?? 'http://localhost',
  // Resolve fetch on every call: openapi-fetch would otherwise keep the
  // fetch that existed at import time and bypass wrappers installed later
  // (MSW in tests and in the mock mode).
  fetch: (request) => globalThis.fetch(request),
})

/** Error of the API with the fields of its error body (see the contract). */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  /** Per-field messages, keyed by body field or query parameter. */
  readonly fields: Record<string, string>

  constructor(status: number, body?: Partial<ErrorBody>, options?: ErrorOptions) {
    const e = body?.error
    super(e?.message ?? 'Не удалось связаться с сервером. Попробуйте ещё раз', options)
    this.name = 'ApiError'
    this.status = status
    this.code = e?.code ?? 'network'
    this.fields = e?.fields ?? {}
  }
}

type Result<T> = { data?: T; error?: unknown; response: Response }

/** Returns the data of a successful response or throws ApiError. */
export async function unwrap<T>(call: Promise<Result<T>>): Promise<T> {
  let result: Result<T>
  try {
    result = await call
  } catch (cause) {
    // fetch itself failed: offline, DNS, CORS, aborted. Keep the cause for debugging.
    throw new ApiError(0, undefined, { cause })
  }
  if (result.error !== undefined || !result.response.ok) {
    throw new ApiError(result.response.status, result.error as Partial<ErrorBody> | undefined)
  }
  return result.data as T
}
