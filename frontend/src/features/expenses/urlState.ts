// The expenses screen state lives in the URL query, so a filtered view
// survives a reload and can be shared as a link. Defaults are not written:
// a clean URL means "this month, newest first, page 1".
import { isValid } from 'date-fns'
import { useCallback, useSyncExternalStore } from 'react'

import type { SortOrder } from '@/api/hooks'
import { isCategoryCode, type CategoryCode } from '@/shared/categories'
import { fromISODate, presetPeriod, toISODate, type ISODate, type Period } from '@/shared/dates'

/** Columns the table can be sorted by (design: date and amount). */
export type SortKey = 'spent_at' | 'amount'

export type ExpensesView = {
  userId?: number
  from: ISODate
  to: ISODate
  category?: CategoryCode
  sort: SortKey
  order: SortOrder
  page: number
}

export const DEFAULT_SORT: SortKey = 'spent_at'
export const DEFAULT_ORDER: SortOrder = 'desc'

export const defaultPeriod = (now = new Date()): Period => presetPeriod('month', now)

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/

/** A real calendar date in yyyy-MM-dd (rejects 2026-02-31), time zone independent. */
function validDate(s: string | null): s is ISODate {
  if (!s || !DATE_RE.test(s)) return false
  const d = fromISODate(s)
  return isValid(d) && toISODate(d) === s
}

/** Reads the view from a query string; anything invalid falls back to the default. */
export function parseView(search: string, now = new Date()): ExpensesView {
  const q = new URLSearchParams(search)
  const view: ExpensesView = { ...defaultPeriod(now), sort: DEFAULT_SORT, order: DEFAULT_ORDER, page: 1 }

  const userId = q.get('user_id')
  if (userId && /^\d{1,15}$/.test(userId)) view.userId = Number(userId)

  const from = q.get('from')
  const to = q.get('to')
  if (validDate(from) && validDate(to) && from <= to) {
    view.from = from
    view.to = to
  }

  const category = q.get('category')
  if (category && isCategoryCode(category)) view.category = category

  const sort = q.get('sort')
  if (sort === 'spent_at' || sort === 'amount') view.sort = sort
  const order = q.get('order')
  if (order === 'asc' || order === 'desc') view.order = order

  const page = q.get('page')
  if (page && /^\d{1,6}$/.test(page) && Number(page) >= 1) view.page = Number(page)
  return view
}

/** Writes the view as a query string without the default values. */
export function serializeView(v: ExpensesView, now = new Date()): string {
  const q = new URLSearchParams()
  if (v.userId !== undefined) q.set('user_id', String(v.userId))
  const def = defaultPeriod(now)
  if (v.from !== def.from || v.to !== def.to) {
    q.set('from', v.from)
    q.set('to', v.to)
  }
  if (v.category) q.set('category', v.category)
  if (v.sort !== DEFAULT_SORT) q.set('sort', v.sort)
  if (v.order !== DEFAULT_ORDER) q.set('order', v.order)
  if (v.page > 1) q.set('page', String(v.page))
  const s = q.toString()
  return s ? `?${s}` : ''
}

/** True when the filters (not sorting or paging) are the defaults. */
export function hasDefaultFilters(v: ExpensesView, now = new Date()): boolean {
  const def = defaultPeriod(now)
  return v.userId === undefined && v.category === undefined && v.from === def.from && v.to === def.to
}

export type ViewPatch = Partial<ExpensesView>

/** Applies a change; any change other than the page itself goes back to page 1. */
export function applyPatch(v: ExpensesView, patch: ViewPatch): ExpensesView {
  const next = { ...v, ...patch }
  if (!('page' in patch)) next.page = 1
  // Explicit undefined clears optional filters.
  if ('userId' in patch && patch.userId === undefined) delete next.userId
  if ('category' in patch && patch.category === undefined) delete next.category
  return next
}

const URL_EVENT = 'expenses:urlchange'

function subscribe(onChange: () => void) {
  window.addEventListener('popstate', onChange)
  window.addEventListener(URL_EVENT, onChange)
  return () => {
    window.removeEventListener('popstate', onChange)
    window.removeEventListener(URL_EVENT, onChange)
  }
}

const getSearch = () => window.location.search

/**
 * The expenses view bound to the URL. Updates replace the history entry
 * (filter tweaks should not flood the back button).
 */
export function useExpensesView(): [ExpensesView, (patch: ViewPatch) => void] {
  const search = useSyncExternalStore(subscribe, getSearch)
  const view = parseView(search)
  const update = useCallback((patch: ViewPatch) => {
    const next = applyPatch(parseView(window.location.search), patch)
    const url = window.location.pathname + serializeView(next) + window.location.hash
    window.history.replaceState(window.history.state, '', url)
    window.dispatchEvent(new Event(URL_EVENT))
  }, [])
  return [view, update]
}
