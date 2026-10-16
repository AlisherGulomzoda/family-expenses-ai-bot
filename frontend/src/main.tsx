import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { App } from '@/App'
import { Providers } from '@/app/Providers'
import './index.css'

async function main() {
  if (import.meta.env.VITE_API_MOCKS === 'true') {
    // Loaded only in this mode, so MSW never reaches the production bundle.
    const { startMockApi } = await import('@/mocks/browser')
    await startMockApi()
  }

  const root = document.getElementById('root')
  if (!root) {
    throw new Error('#root element is missing in index.html')
  }
  createRoot(root).render(
    <StrictMode>
      <Providers>
        <App />
      </Providers>
    </StrictMode>,
  )
}

void main()
