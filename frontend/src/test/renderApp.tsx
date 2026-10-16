import { render } from '@testing-library/react'

import { App } from '@/App'
import { Providers } from '@/app/Providers'
import { createQueryClient } from '@/app/queryClient'

// The page loads the form, the calendar and the phone drawer lazily (features/expenses/lazy.tsx).
// Their first import in a test file takes longer than findBy waits, so load them with this helper.
await Promise.all([
  import('@/features/expenses/ExpenseFormDialog'),
  import('@/features/expenses/PeriodCalendar'),
  import('@/features/expenses/FiltersDrawer'),
])

/** Renders the whole app at the given query string, without retries. */
export function renderApp(search = '') {
  window.history.replaceState(null, '', `/${search}`)
  const client = createQueryClient()
  client.setDefaultOptions({ queries: { retry: false }, mutations: { retry: false } })
  return render(
    <Providers queryClient={client}>
      <App />
    </Providers>,
  )
}
