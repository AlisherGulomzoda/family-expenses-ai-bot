import { useState } from 'react'

import { toISODate, type Period } from '@/shared/dates'

/**
 * Two clicks pick a period: the first starts a new range, the second ends it
 * (the same day twice is a one-day period). react-day-picker would instead
 * stretch the current range, so the clicks are interpreted here.
 */
export function useRangeClicks(onPick: (p: Period) => void) {
  const [start, setStart] = useState<Date>()
  const click = (day: Date) => {
    if (!start) {
      setStart(day)
      return
    }
    const [from, to] = start <= day ? [start, day] : [day, start]
    setStart(undefined)
    onPick({ from: toISODate(from), to: toISODate(to) })
  }
  return { start, click, reset: () => setStart(undefined) }
}
