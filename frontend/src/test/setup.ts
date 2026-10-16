import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { toast } from 'sonner'
import { afterAll, afterEach, beforeAll } from 'vitest'

import { server } from '@/mocks/node'

import { installMatchMedia, resetViewport } from './viewport'

// Network calls in tests go to MSW; an unmocked request fails the test.
beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))
afterAll(() => server.close())

afterEach(() => {
  cleanup()
  // Sonner keeps toasts in a module-level store: drop them between tests.
  toast.dismiss()
  server.resetHandlers()
  localStorage.clear()
  document.documentElement.className = ''
  resetViewport()
})

// jsdom has no matchMedia: width queries follow the test viewport (desktop by
// default, see setViewport); prefers-color-scheme reports "light".
installMatchMedia()

// Radix primitives observe their size.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.ResizeObserver ??= ResizeObserverStub as unknown as typeof ResizeObserver

// Radix Select and Popover use pointer capture and scrolling, which jsdom lacks.
Element.prototype.hasPointerCapture ??= () => false
Element.prototype.setPointerCapture ??= () => {}
Element.prototype.releasePointerCapture ??= () => {}
Element.prototype.scrollIntoView ??= () => {}
