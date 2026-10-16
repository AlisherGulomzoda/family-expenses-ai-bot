// Category map per design-spec §2: one object for the whole app.
// Classes are written in full, otherwise Tailwind does not see them.
import { Car, HeartPulse, House, Shapes, ShoppingBag, Ticket, Utensils, type LucideIcon } from 'lucide-react'

import type { components } from '@/api/schema'

export type CategoryCode = components['schemas']['CategoryCode']

type CategoryStyle = {
  label: string
  icon: LucideIcon
  /** CSS color for charts */
  chart: string
  /** badge and icon circle: text on a soft background */
  badge: string
  /** bars, sectors, dots */
  fill: string
}

// The order is fixed: filters, selects and tiles follow it; only charts sort by amount.
// `satisfies Record<CategoryCode, ...>` fails the build if the API adds or renames a category.
export const CATEGORIES = {
  food: {
    label: 'Еда',
    icon: Utensils,
    chart: 'var(--chart-1)',
    badge: 'bg-chart-1-soft text-chart-1-ink',
    fill: 'bg-chart-1',
  },
  transport: {
    label: 'Транспорт',
    icon: Car,
    chart: 'var(--chart-2)',
    badge: 'bg-chart-2-soft text-chart-2-ink',
    fill: 'bg-chart-2',
  },
  housing: {
    label: 'Жильё',
    icon: House,
    chart: 'var(--chart-3)',
    badge: 'bg-chart-3-soft text-chart-3-ink',
    fill: 'bg-chart-3',
  },
  entertainment: {
    label: 'Развлечения',
    icon: Ticket,
    chart: 'var(--chart-4)',
    badge: 'bg-chart-4-soft text-chart-4-ink',
    fill: 'bg-chart-4',
  },
  health: {
    label: 'Здоровье',
    icon: HeartPulse,
    chart: 'var(--chart-5)',
    badge: 'bg-chart-5-soft text-chart-5-ink',
    fill: 'bg-chart-5',
  },
  shopping: {
    label: 'Покупки',
    icon: ShoppingBag,
    chart: 'var(--chart-6)',
    badge: 'bg-chart-6-soft text-chart-6-ink',
    fill: 'bg-chart-6',
  },
  other: {
    label: 'Другое',
    icon: Shapes,
    chart: 'var(--chart-7)',
    badge: 'bg-chart-7-soft text-chart-7-ink',
    fill: 'bg-chart-7',
  },
} as const satisfies Record<CategoryCode, CategoryStyle>

export const CATEGORY_CODES = Object.keys(CATEGORIES) as CategoryCode[]

export const isCategoryCode = (s: string): s is CategoryCode => Object.hasOwn(CATEGORIES, s)
