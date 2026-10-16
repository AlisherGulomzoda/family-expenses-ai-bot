// A controllable matchMedia for jsdom: width-based queries follow the
// "viewport" width set by tests; everything else (e.g. prefers-color-scheme)
// does not match. Tests run at desktop width unless they call setViewport.

type Listener = () => void

let width = 1280
const listeners = new Set<{ query: string; listener: Listener }>()

function matches(query: string): boolean {
  const min = /\(min-width:\s*(\d+)px\)/.exec(query)
  const max = /\(max-width:\s*(\d+)px\)/.exec(query)
  if (!min && !max) return false
  return (!min || width >= Number(min[1])) && (!max || width <= Number(max[1]))
}

export function installMatchMedia() {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string): MediaQueryList =>
      ({
        get matches() {
          return matches(query)
        },
        media: query,
        onchange: null,
        addEventListener: (_: string, listener: Listener) => listeners.add({ query, listener }),
        removeEventListener: (_: string, listener: Listener) => {
          for (const l of listeners) if (l.listener === listener) listeners.delete(l)
        },
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
      }) as unknown as MediaQueryList,
  })
}

/** Changes the viewport width and notifies the media queries that flipped. */
export function setViewport(next: number) {
  const before = new Map([...listeners].map((l) => [l, matches(l.query)]))
  width = next
  for (const [l, was] of before) if (matches(l.query) !== was) l.listener()
}

export const PHONE = 390
export const DESKTOP = 1280

export function resetViewport() {
  width = DESKTOP
}
