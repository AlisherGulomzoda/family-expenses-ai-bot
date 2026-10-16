import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { App } from '@/App'
import { Providers } from '@/app/Providers'
import './index.css'

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
