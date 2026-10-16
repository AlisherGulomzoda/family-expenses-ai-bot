import { keepPreviousData, useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api, unwrap, type Expense, type ExpenseInput } from '@/api/client'
import type { CategoryCode } from '@/shared/categories'

/** Ten expenses per page / per "show more" (design-spec §4). */
export const PAGE_SIZE = 10

export type SortField = 'spent_at' | 'amount' | 'created_at'
export type SortOrder = 'asc' | 'desc'

/** What the expenses screen filters by; mirrors the URL query. */
export type ExpenseFilters = {
  userId?: number
  from?: string
  to?: string
  category?: CategoryCode
}

export type ListParams = ExpenseFilters & {
  sort?: SortField
  order?: SortOrder
  page?: number // 1-based
}

/** Query keys in one place: every expense query starts with ['expenses']. */
export const queryKeys = {
  users: ['users'] as const,
  expenses: ['expenses'] as const,
  list: (p: ListParams) => ['expenses', 'list', p] as const,
  infinite: (p: ListParams) => ['expenses', 'infinite', p] as const,
  summary: (f: ExpenseFilters) => ['expenses', 'summary', f] as const,
}

function filterQuery(f: ExpenseFilters) {
  return { user_id: f.userId, from: f.from, to: f.to, category: f.category }
}

export function useUsers() {
  return useQuery({
    queryKey: queryKeys.users,
    queryFn: ({ signal }) => unwrap(api.GET('/api/users', { signal })),
    staleTime: 5 * 60_000,
  })
}

export function useExpenses(p: ListParams) {
  const page = p.page ?? 1
  return useQuery({
    queryKey: queryKeys.list(p),
    queryFn: ({ signal }) =>
      unwrap(
        api.GET('/api/expenses', {
          params: {
            query: {
              ...filterQuery(p),
              sort: p.sort,
              order: p.order,
              limit: PAGE_SIZE,
              offset: (page - 1) * PAGE_SIZE,
            },
          },
          signal,
        }),
      ),
    // The table keeps showing the previous page while the next one loads.
    placeholderData: keepPreviousData,
  })
}

/** Mobile list: "Показать ещё" loads the next ten. */
export function useExpensesInfinite(p: Omit<ListParams, 'page'>) {
  return useInfiniteQuery({
    queryKey: queryKeys.infinite(p),
    initialPageParam: 0,
    queryFn: ({ pageParam, signal }) =>
      unwrap(
        api.GET('/api/expenses', {
          params: {
            query: { ...filterQuery(p), sort: p.sort, order: p.order, limit: PAGE_SIZE, offset: pageParam },
          },
          signal,
        }),
      ),
    getNextPageParam: (last, pages) => {
      const loaded = pages.reduce((n, pg) => n + pg.items.length, 0)
      return loaded < last.total ? loaded : undefined
    },
  })
}

export function useSummary(f: ExpenseFilters) {
  return useQuery({
    queryKey: queryKeys.summary(f),
    queryFn: ({ signal }) => unwrap(api.GET('/api/expenses/summary', { params: { query: filterQuery(f) }, signal })),
    placeholderData: keepPreviousData,
  })
}

/** After any change, lists and totals are refetched. */
function useInvalidateExpenses() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: queryKeys.expenses })
}

export function useCreateExpense() {
  const invalidate = useInvalidateExpenses()
  return useMutation({
    mutationFn: (body: ExpenseInput) => unwrap(api.POST('/api/expenses', { body })),
    onSuccess: invalidate,
  })
}

export function useUpdateExpense() {
  const invalidate = useInvalidateExpenses()
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: ExpenseInput }) =>
      unwrap(api.PUT('/api/expenses/{id}', { params: { path: { id } }, body })),
    onSuccess: invalidate,
  })
}

export function useDeleteExpense() {
  const invalidate = useInvalidateExpenses()
  return useMutation({
    mutationFn: async (expense: Pick<Expense, 'id'>) => {
      await unwrap(api.DELETE('/api/expenses/{id}', { params: { path: { id: expense.id } } }))
      return expense
    },
    onSuccess: invalidate,
  })
}
