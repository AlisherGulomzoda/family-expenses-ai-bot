// Dates per design-spec §1: "dd.MM.yyyy" in the table, "d MMM" on mobile cards.
// The API uses "yyyy-MM-dd" strings; they are handled as local calendar dates.
import {
  addDays,
  endOfMonth,
  endOfWeek,
  format,
  isSameDay,
  isSameMonth,
  isSameYear,
  parseISO,
  startOfMonth,
  startOfWeek,
  subMonths,
} from 'date-fns'

/** API date string, yyyy-MM-dd. */
export type ISODate = string

export const toISODate = (d: Date): ISODate => format(d, 'yyyy-MM-dd')

export const fromISODate = (s: ISODate): Date => parseISO(s)

// The design abbreviates months to three letters without a dot ("30 сен",
// "15 окт"); date-fns uses dictionary forms ("сент.", "нояб."), hence our own.
const SHORT_MONTHS = ['янв', 'фев', 'мар', 'апр', 'мая', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек']

const dayMonth = (d: Date) => `${d.getDate()} ${SHORT_MONTHS[d.getMonth()]}`
const dayMonthYear = (d: Date) => `${dayMonth(d)} ${d.getFullYear()}`

/** 15.10.2026 */
export const formatDate = (s: ISODate): string => format(fromISODate(s), 'dd.MM.yyyy')

/** "30 сен" */
export const formatShortDate = (s: ISODate): string => dayMonth(fromISODate(s))

/** "Сегодня, 16.10.2026", "Вчера, 15.10.2026" or "14.10.2026". */
export function formatDayLabel(s: ISODate, now = new Date()): string {
  const d = fromISODate(s)
  if (isSameDay(d, now)) return `Сегодня, ${formatDate(s)}`
  if (isSameDay(d, addDays(now, -1))) return `Вчера, ${formatDate(s)}`
  return formatDate(s)
}

export type Period = { from: ISODate; to: ISODate }

export type PeriodPreset = 'today' | 'week' | 'month' | 'prev-month'

export const PERIOD_PRESETS: { id: PeriodPreset; label: string }[] = [
  { id: 'today', label: 'Сегодня' },
  { id: 'week', label: 'Неделя' },
  { id: 'month', label: 'Этот месяц' },
  { id: 'prev-month', label: 'Прошлый месяц' },
]

/** Bounds of a quick period; weeks start on Monday, as in Russia. */
export function presetPeriod(preset: PeriodPreset, now = new Date()): Period {
  const iso = (from: Date, to: Date) => ({ from: toISODate(from), to: toISODate(to) })
  switch (preset) {
    case 'today':
      return iso(now, now)
    case 'week':
      return iso(startOfWeek(now, { weekStartsOn: 1 }), endOfWeek(now, { weekStartsOn: 1 }))
    case 'month':
      return iso(startOfMonth(now), endOfMonth(now))
    case 'prev-month': {
      const prev = subMonths(now, 1)
      return iso(startOfMonth(prev), endOfMonth(prev))
    }
  }
}

/** The preset a period matches, if any. */
export function matchPreset(p: Partial<Period>, now = new Date()): PeriodPreset | undefined {
  return PERIOD_PRESETS.map((x) => x.id).find((id) => {
    const q = presetPeriod(id, now)
    return q.from === p.from && q.to === p.to
  })
}

/** "1–31 окт 2026", "28 сен – 4 окт 2026", "15 окт 2026". */
export function formatPeriod(p: Period): string {
  const from = fromISODate(p.from)
  const to = fromISODate(p.to)
  if (isSameDay(from, to)) return dayMonthYear(to)
  if (isSameMonth(from, to)) return `${from.getDate()}–${dayMonthYear(to)}`
  if (isSameYear(from, to)) return `${dayMonth(from)} – ${dayMonthYear(to)}`
  return `${dayMonthYear(from)} – ${dayMonthYear(to)}`
}
