import { render } from '@testing-library/react'

import { App } from '@/App'
import { Providers } from '@/app/Providers'
import { createQueryClient } from '@/app/queryClient'

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
