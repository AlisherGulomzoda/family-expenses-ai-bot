// The expense form: field rules, defaults and the conversion to the API body.
// Messages and limits match the backend (expense.Validate) and openapi.yaml.
import { addDays } from 'date-fns'
import { z } from 'zod'

import type { Expense, ExpenseInput, User } from '@/api/client'
import { isCategoryCode, type CategoryCode } from '@/shared/categories'
import { toISODate, type ISODate } from '@/shared/dates'
import { kopecksToRubles, parseRubInput, rublesToInput } from '@/shared/money'

const MAX_KOPECKS = 100_000_000 * 100
const DATE_RE = /^\d{4}-\d{2}-\d{2}$/

/** Form values are what the inputs hold: strings until they are submitted. */
export type FormValues = {
  userId: string
  amount: string
  category: CategoryCode | ''
  spentAt: ISODate
  comment: string
}

export const formSchema = z.object({
  userId: z.string().min(1, 'Выберите члена семьи'),
  amount: z.string().superRefine((text, ctx) => {
    if (text.trim() === '') {
      ctx.addIssue({ code: 'custom', message: 'Укажите сумму' })
      return
    }
    const kopecks = parseRubInput(text)
    if (kopecks === null) {
      ctx.addIssue({ code: 'custom', message: 'Сумма — число, не больше двух знаков после запятой' })
    } else if (kopecks <= 0) {
      ctx.addIssue({ code: 'custom', message: 'Укажите сумму больше нуля' })
    } else if (kopecks > MAX_KOPECKS) {
      ctx.addIssue({ code: 'custom', message: 'Сумма не может быть больше 100 000 000 ₽' })
    }
  }),
  category: z.string().refine(isCategoryCode, 'Выберите категорию'),
  spentAt: z
    .string()
    .regex(DATE_RE, 'Укажите дату')
    // Tomorrow is allowed, as on the backend: a family member may be in another time zone.
    .refine((d) => d <= toISODate(addDays(new Date(), 1)), 'Дата не может быть в будущем'),
  comment: z.string().max(500, 'Комментарий не длиннее 500 символов'),
})

/** Values for a new expense. */
export function newValues(userId: number | undefined): FormValues {
  return {
    userId: userId === undefined ? '' : String(userId),
    amount: '',
    category: '',
    spentAt: toISODate(new Date()),
    comment: '',
  }
}

/** Values of an existing expense. */
export function editValues(e: Expense): FormValues {
  return {
    userId: String(e.user_id),
    amount: rublesToInput(e.amount),
    category: e.category,
    spentAt: e.spent_at,
    comment: e.comment ?? '',
  }
}

/** Valid form values to the API body (PUT replaces every field). */
export function toBody(v: FormValues): ExpenseInput {
  const kopecks = parseRubInput(v.amount)
  if (kopecks === null || !isCategoryCode(v.category)) {
    throw new Error('toBody needs validated values')
  }
  const comment = v.comment.trim()
  return {
    user_id: Number(v.userId),
    amount: kopecksToRubles(kopecks),
    category: v.category,
    spent_at: v.spentAt,
    comment: comment === '' ? null : comment,
  }
}

/** API field names (422 `fields`) to form field names. */
export const API_FIELDS: Record<string, keyof FormValues> = {
  user_id: 'userId',
  amount: 'amount',
  category: 'category',
  spent_at: 'spentAt',
  comment: 'comment',
}

// The member chosen last time; used when the filter does not name one.
const LAST_MEMBER_KEY = 'expenses.lastMemberId'

export function rememberMember(id: number) {
  try {
    localStorage.setItem(LAST_MEMBER_KEY, String(id))
  } catch {
    // Private mode or blocked storage: the default just falls back to the first member.
  }
}

/** Default member of a new expense: the filter's, else the last chosen, else the first. */
export function defaultMember(users: User[] | undefined, filterUserId?: number): number | undefined {
  if (filterUserId !== undefined) return filterUserId
  let last: number | undefined
  try {
    const raw = localStorage.getItem(LAST_MEMBER_KEY)
    last = raw ? Number(raw) : undefined
  } catch {
    last = undefined
  }
  if (last !== undefined && users?.some((u) => u.id === last)) return last
  return users?.[0]?.id
}
