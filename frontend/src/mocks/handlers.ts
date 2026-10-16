// MSW handlers that behave like the real API (backend/api/openapi.yaml):
// inclusive period filters, sorting with id as the tie-breaker, pagination,
// totals over the whole filter, and the same 400/404/422 error shapes.
import { http, HttpResponse, type HttpHandler } from 'msw'

import type { Expense, ExpenseInput } from '@/api/client'
import { CATEGORIES, CATEGORY_CODES, isCategoryCode, type CategoryCode } from '@/shared/categories'
import { toISODate } from '@/shared/dates'

import type { MockDb } from './db'

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/

type Fields = Record<string, string>

const fail = (status: number, code: string, message: string, fields?: Fields) =>
  HttpResponse.json({ error: { code, message, ...(fields && { fields }) } }, { status })

const badRequest = (fields: Fields) => fail(400, 'bad_request', 'Некорректные параметры запроса', fields)
const notFound = () => fail(404, 'not_found', 'Трата не найдена')
const invalid = (fields: Fields) => fail(422, 'validation_error', 'Проверьте заполнение полей', fields)

type Filter = { userId?: number; from?: string; to?: string; category?: CategoryCode }

function parseFilter(q: URLSearchParams): { filter: Filter; fields: Fields } {
  const filter: Filter = {}
  const fields: Fields = {}
  const userId = q.get('user_id')
  if (userId) {
    if (/^\d+$/.test(userId)) filter.userId = Number(userId)
    else fields.user_id = 'Ожидается Telegram ID (целое число)'
  }
  for (const key of ['from', 'to'] as const) {
    const v = q.get(key)
    if (!v) continue
    if (DATE_RE.test(v)) filter[key] = v
    else fields[key] = 'Ожидается дата в формате ГГГГ-ММ-ДД'
  }
  const category = q.get('category')
  if (category) {
    if (isCategoryCode(category)) filter.category = category
    else fields.category = 'Неизвестная категория'
  }
  if (filter.from && filter.to && filter.from > filter.to) fields.from = 'Начало периода позже его конца'
  return { filter, fields }
}

const matches = (e: Expense, f: Filter) =>
  (f.userId === undefined || e.user_id === f.userId) &&
  (f.from === undefined || e.spent_at >= f.from) &&
  (f.to === undefined || e.spent_at <= f.to) &&
  (f.category === undefined || e.category === f.category)

const round2 = (n: number) => Math.round(n * 100) / 100

function validateInput(db: MockDb, raw: unknown): { input?: ExpenseInput; fields?: Fields } {
  const b = (raw ?? {}) as Partial<Record<keyof ExpenseInput, unknown>>
  const fields: Fields = {}
  if (typeof b.user_id !== 'number') fields.user_id = 'Выберите члена семьи'
  else if (!db.users.some((u) => u.id === b.user_id)) fields.user_id = 'Неизвестный член семьи'
  if (b.amount === undefined || b.amount === null) fields.amount = 'Укажите сумму'
  else if (typeof b.amount !== 'number' || !/^\d+(\.\d{1,2})?$/.test(String(Math.abs(b.amount))))
    fields.amount = 'Сумма — число, не больше двух знаков после запятой'
  else if (b.amount <= 0) fields.amount = 'Укажите сумму больше нуля'
  else if (b.amount > 100_000_000) fields.amount = 'Сумма не может быть больше 100 000 000 ₽'
  if (typeof b.category !== 'string' || !isCategoryCode(b.category)) fields.category = 'Выберите категорию'
  if (typeof b.spent_at !== 'string') fields.spent_at = 'Укажите дату'
  else if (!DATE_RE.test(b.spent_at)) fields.spent_at = 'Дата в формате ГГГГ-ММ-ДД'
  else if (b.spent_at > toISODate(new Date(db.now().getTime() + 86_400_000)))
    fields.spent_at = 'Дата не может быть в будущем'
  const comment = typeof b.comment === 'string' ? b.comment.trim() : null
  if (comment && [...comment].length > 500) fields.comment = 'Комментарий не длиннее 500 символов'
  if (Object.keys(fields).length > 0) return { fields }
  return {
    input: {
      user_id: b.user_id as number,
      amount: b.amount as number,
      category: b.category as CategoryCode,
      spent_at: b.spent_at as string,
      comment: comment || null,
    },
  }
}

export function createHandlers(db: MockDb): HttpHandler[] {
  const userName = (id: number) => db.users.find((u) => u.id === id)?.name ?? ''
  const findExpense = (raw: string | readonly string[] | undefined) => {
    const id = Number(raw)
    return Number.isInteger(id) && id > 0 ? { id, expense: db.expenses.find((e) => e.id === id) } : undefined
  }

  return [
    http.get('*/api/users', () =>
      HttpResponse.json([...db.users].sort((a, b) => a.name.localeCompare(b.name, 'ru') || a.id - b.id)),
    ),

    http.get('*/api/categories', () =>
      HttpResponse.json(CATEGORY_CODES.map((code) => ({ code, name: CATEGORIES[code].label }))),
    ),

    http.get('*/api/expenses/summary', ({ request }) => {
      const { filter, fields } = parseFilter(new URL(request.url).searchParams)
      if (Object.keys(fields).length > 0) return badRequest(fields)
      const rows = db.expenses.filter((e) => matches(e, filter))
      const byCategory = CATEGORY_CODES.map((category) => {
        const own = rows.filter((e) => e.category === category)
        return { category, total: round2(own.reduce((s, e) => s + e.amount, 0)), count: own.length }
      })
        .filter((c) => c.count > 0)
        .sort((a, b) => b.total - a.total) // stable: equal totals keep the display order
      return HttpResponse.json({
        total: round2(rows.reduce((s, e) => s + e.amount, 0)),
        count: rows.length,
        by_category: byCategory,
      })
    }),

    http.get('*/api/expenses', ({ request }) => {
      const q = new URL(request.url).searchParams
      const { filter, fields } = parseFilter(q)
      const sort = q.get('sort') ?? 'spent_at'
      const order = q.get('order') ?? 'desc'
      const limit = Number(q.get('limit') ?? 20)
      const offset = Number(q.get('offset') ?? 0)
      if (!['spent_at', 'amount', 'created_at'].includes(sort)) fields.sort = 'Неизвестное поле сортировки'
      if (!['asc', 'desc'].includes(order)) fields.order = 'Порядок сортировки — asc или desc'
      if (!Number.isInteger(limit) || limit < 1 || limit > 100) fields.limit = 'Размер страницы — от 1 до 100'
      if (!Number.isInteger(offset) || offset < 0) fields.offset = 'Смещение не может быть отрицательным'
      if (Object.keys(fields).length > 0) return badRequest(fields)

      const key = sort as 'spent_at' | 'amount' | 'created_at'
      const dir = order === 'asc' ? 1 : -1
      const rows = db.expenses
        .filter((e) => matches(e, filter))
        .sort((a, b) => dir * (a[key] < b[key] ? -1 : a[key] > b[key] ? 1 : a.id - b.id))
      return HttpResponse.json({ items: rows.slice(offset, offset + limit), total: rows.length })
    }),

    http.post('*/api/expenses', async ({ request }) => {
      const { input, fields } = validateInput(db, await request.json().catch(() => null))
      if (!input) return invalid(fields ?? {})
      const now = db.now().toISOString()
      const expense: Expense = {
        id: db.nextId++,
        ...input,
        comment: input.comment ?? null,
        user_name: userName(input.user_id),
        source: 'web',
        created_at: now,
        updated_at: now,
      }
      db.expenses.push(expense)
      return HttpResponse.json(expense, { status: 201, headers: { Location: `/api/expenses/${expense.id}` } })
    }),

    http.get('*/api/expenses/:id', ({ params }) => {
      const found = findExpense(params.id)
      if (!found) return badRequest({ id: 'Идентификатор траты — положительное целое число' })
      return found.expense ? HttpResponse.json(found.expense) : notFound()
    }),

    http.put('*/api/expenses/:id', async ({ params, request }) => {
      const found = findExpense(params.id)
      if (!found) return badRequest({ id: 'Идентификатор траты — положительное целое число' })
      if (!found.expense) return notFound()
      const { input, fields } = validateInput(db, await request.json().catch(() => null))
      if (!input) return invalid(fields ?? {})
      Object.assign(found.expense, input, {
        comment: input.comment ?? null,
        user_name: userName(input.user_id),
        updated_at: db.now().toISOString(),
      })
      return HttpResponse.json(found.expense)
    }),

    http.delete('*/api/expenses/:id', ({ params }) => {
      const found = findExpense(params.id)
      if (!found) return badRequest({ id: 'Идентификатор траты — положительное целое число' })
      if (!found.expense) return notFound()
      db.expenses = db.expenses.filter((e) => e.id !== found.id)
      return new HttpResponse(null, { status: 204 })
    }),
  ]
}
