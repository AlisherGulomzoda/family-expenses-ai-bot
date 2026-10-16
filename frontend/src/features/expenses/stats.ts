import type { Summary } from '@/api/client'

/** Average expense in rubles, rounded to kopecks; 0 when there are none. */
export function average(s: Pick<Summary, 'total' | 'count'>): number {
  return s.count > 0 ? Math.round((s.total / s.count) * 100) / 100 : 0
}

/** Share of a category in the total, whole percent; 0 for an empty total. */
export function share(part: number, total: number): number {
  return total > 0 ? Math.round((part / total) * 100) : 0
}
