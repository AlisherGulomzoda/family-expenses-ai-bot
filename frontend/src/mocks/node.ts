// Mock API for tests: a fresh database per test.
import { setupServer } from 'msw/node'

import { createDb, type MockDb } from './db'
import { createHandlers } from './handlers'

/** Fixed "now" of the test data: Friday 2026-10-16, 10:00. */
export const TEST_NOW = new Date(2026, 9, 16, 10, 0, 0)

export const server = setupServer()

/** Resets the mock API to fresh data and returns its database. */
export function resetMockApi(now: Date = TEST_NOW): MockDb {
  const db = createDb(now)
  server.resetHandlers(...createHandlers(db))
  return db
}
