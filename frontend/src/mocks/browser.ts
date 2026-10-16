// Mock API in the browser for working on the UI without the backend:
//   VITE_API_MOCKS=true npm run dev
import { setupWorker } from 'msw/browser'

import { createDb } from './db'
import { createHandlers } from './handlers'

export async function startMockApi() {
  const worker = setupWorker(...createHandlers(createDb()))
  await worker.start({ onUnhandledFrame: 'bypass', quiet: true })
  console.info('[mocks] API is mocked by MSW (VITE_API_MOCKS=true)')
}
