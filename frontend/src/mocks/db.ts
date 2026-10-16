// In-memory data behind the MSW handlers: realistic expenses of four family
// members over the last ~45 days. Deterministic for a given "now".
import { addDays } from 'date-fns'

import type { Expense, User } from '@/api/client'
import { toISODate } from '@/shared/dates'
import type { CategoryCode } from '@/shared/categories'

export const MOCK_USERS: User[] = [
  { id: 100000001, name: 'Алишер', username: 'alisher' },
  { id: 100000002, name: 'Мадина', username: 'madina_k' },
  { id: 100000003, name: 'Тимур', username: 'timur' },
  { id: 100000004, name: 'Зарина', username: null },
]

const TEMPLATES: { category: CategoryCode; comments: string[]; min: number; max: number }[] = [
  {
    category: 'food',
    comments: ['кофе', 'продукты в «Пятёрочке»', 'обед', 'доставка пиццы', 'рынок'],
    min: 150,
    max: 4200,
  },
  { category: 'transport', comments: ['такси', 'метро', 'бензин', 'такси до работы', 'парковка'], min: 60, max: 2500 },
  { category: 'housing', comments: ['коммуналка', 'интернет', 'ремонт крана'], min: 600, max: 9000 },
  { category: 'entertainment', comments: ['кино', 'концерт', 'подписка', 'боулинг'], min: 300, max: 3500 },
  { category: 'health', comments: ['аптека', 'стоматолог', 'анализы'], min: 250, max: 6000 },
  { category: 'shopping', comments: ['кроссовки', 'футболка', 'подарок маме', 'наушники'], min: 900, max: 12000 },
  { category: 'other', comments: ['подарок коллеге', 'стрижка', 'ремонт обуви'], min: 200, max: 2000 },
]

/** mulberry32: a tiny seeded PRNG, so the mock data is stable between reloads. */
function prng(seed: number) {
  let a = seed
  return () => {
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

export type MockDb = {
  users: User[]
  expenses: Expense[]
  nextId: number
  now: () => Date
}

/** Creates the mock database; `count` expenses spread over the last 45 days. */
export function createDb(now: Date = new Date(), count = 42): MockDb {
  const rand = prng(20261016)
  const pick = <T>(xs: T[]): T => xs[Math.floor(rand() * xs.length)] as T
  const expenses: Expense[] = []
  for (let i = 0; i < count; i++) {
    const t = pick(TEMPLATES)
    const user = pick(MOCK_USERS)
    const spent = addDays(now, -Math.floor(rand() * 45))
    // Mostly whole rubles, sometimes with kopecks.
    const rubles = Math.round(t.min + rand() * (t.max - t.min))
    const amount = rand() < 0.2 ? rubles + Math.round(rand() * 99) / 100 : rubles
    const created = new Date(spent.getTime() + i * 60_000).toISOString()
    expenses.push({
      id: i + 1,
      user_id: user.id,
      user_name: user.name,
      amount,
      category: t.category,
      spent_at: toISODate(spent),
      comment: rand() < 0.1 ? null : pick(t.comments),
      source: rand() < 0.8 ? 'bot' : 'web',
      created_at: created,
      updated_at: created,
    })
  }
  return { users: structuredClone(MOCK_USERS), expenses, nextId: count + 1, now: () => now }
}
