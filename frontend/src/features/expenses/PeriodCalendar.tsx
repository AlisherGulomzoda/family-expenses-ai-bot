import { ru } from 'react-day-picker/locale'

import { Calendar } from '@/components/ui/calendar'
import { fromISODate, type Period } from '@/shared/dates'

import { useRangeClicks } from './useRangeClicks'

/** Calendar that picks a period with two clicks (see useRangeClicks). */
export function PeriodCalendar({
  period,
  onPick,
  numberOfMonths = 1,
  className,
}: {
  period: Period
  onPick: (p: Period) => void
  numberOfMonths?: number
  className?: string
}) {
  const range = useRangeClicks(onPick)
  const current = { from: fromISODate(period.from), to: fromISODate(period.to) }
  return (
    <Calendar
      mode="range"
      numberOfMonths={numberOfMonths}
      locale={ru}
      defaultMonth={current.from}
      className={className}
      // `selected` only highlights; the clicks themselves are interpreted by useRangeClicks.
      selected={range.start ? { from: range.start, to: range.start } : current}
      onDayClick={(day, modifiers) => !modifiers.disabled && range.click(day)}
    />
  )
}
