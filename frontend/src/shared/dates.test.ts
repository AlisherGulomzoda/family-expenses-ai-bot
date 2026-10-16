import {
  formatDate,
  formatDayLabel,
  formatPeriod,
  formatShortDate,
  matchPreset,
  presetPeriod,
  type PeriodPreset,
} from '@/shared/dates'

// Friday 2026-10-16, 10:00 local time.
const now = new Date(2026, 9, 16, 10, 0, 0)

describe('formatting', () => {
  it('formats table and card dates', () => {
    expect(formatDate('2026-10-05')).toBe('05.10.2026')
    expect(formatShortDate('2026-09-30')).toBe('30 сен')
    expect(formatShortDate('2026-05-01')).toBe('1 мая')
  })

  it('labels today and yesterday', () => {
    expect(formatDayLabel('2026-10-16', now)).toBe('Сегодня, 16.10.2026')
    expect(formatDayLabel('2026-10-15', now)).toBe('Вчера, 15.10.2026')
    expect(formatDayLabel('2026-10-14', now)).toBe('14.10.2026')
  })

  it.each([
    [{ from: '2026-10-01', to: '2026-10-31' }, '1–31 окт 2026'],
    [{ from: '2026-09-28', to: '2026-10-04' }, '28 сен – 4 окт 2026'],
    [{ from: '2026-10-15', to: '2026-10-15' }, '15 окт 2026'],
    [{ from: '2025-12-29', to: '2026-01-04' }, '29 дек 2025 – 4 янв 2026'],
  ])('formatPeriod(%j) = %s', (p, want) => {
    expect(formatPeriod(p)).toBe(want)
  })
})

describe('quick periods', () => {
  it.each<[PeriodPreset, string, string]>([
    ['today', '2026-10-16', '2026-10-16'],
    ['week', '2026-10-12', '2026-10-18'], // Monday to Sunday
    ['month', '2026-10-01', '2026-10-31'],
    ['prev-month', '2026-09-01', '2026-09-30'],
  ])('%s', (preset, from, to) => {
    expect(presetPeriod(preset, now)).toEqual({ from, to })
  })

  it('handles the turn of the year and Sundays', () => {
    expect(presetPeriod('prev-month', new Date(2026, 0, 10))).toEqual({ from: '2025-12-01', to: '2025-12-31' })
    expect(presetPeriod('week', new Date(2026, 9, 18))).toEqual({ from: '2026-10-12', to: '2026-10-18' })
  })

  it('recognizes a preset from its bounds', () => {
    expect(matchPreset({ from: '2026-10-01', to: '2026-10-31' }, now)).toBe('month')
    expect(matchPreset({ from: '2026-10-02', to: '2026-10-31' }, now)).toBeUndefined()
  })
})
