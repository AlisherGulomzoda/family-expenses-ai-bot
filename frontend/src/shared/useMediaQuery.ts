import { useSyncExternalStore } from 'react'

/** Live result of a CSS media query. */
export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const mql = window.matchMedia(query)
      mql.addEventListener('change', onChange)
      return () => mql.removeEventListener('change', onChange)
    },
    () => window.matchMedia(query).matches,
  )
}

/**
 * The design's main breakpoint (design-spec §5): from 768px (Tailwind `md`)
 * the table, the filter bar and the centered dialog; below it cards, the
 * filter drawer and a full-screen form. Only one variant is rendered, so
 * the phone does not load the table pages and vice versa.
 */
export const DESKTOP_QUERY = '(min-width: 768px)'

export function useIsDesktop(): boolean {
  return useMediaQuery(DESKTOP_QUERY)
}
