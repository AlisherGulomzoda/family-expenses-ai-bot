import { QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'

import { ApiError } from '@/api/client'
import {
  PAGE_SIZE,
  useCreateExpense,
  useDeleteExpense,
  useExpenses,
  useExpensesInfinite,
  useSummary,
  useUpdateExpense,
  useUsers,
} from '@/api/hooks'
import { createQueryClient } from '@/app/queryClient'
import { resetMockApi, server } from '@/mocks/node'

function setup() {
  const db = resetMockApi()
  const client = createQueryClient()
  client.setDefaultOptions({ queries: { retry: false }, mutations: { retry: false } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { db, wrapper }
}

describe('queries', () => {
  it('loads family members sorted by name', async () => {
    const { wrapper } = setup()
    const { result } = renderHook(() => useUsers(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.map((u) => u.name)).toEqual(['Алишер', 'Зарина', 'Мадина', 'Тимур'])
  })

  it('pages and filters expenses on the server', async () => {
    const { db, wrapper } = setup()
    const { result } = renderHook(() => useExpenses({ page: 2, sort: 'amount', order: 'desc' }), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const all = [...db.expenses].sort((a, b) => b.amount - a.amount || b.id - a.id)
    expect(result.current.data?.total).toBe(db.expenses.length)
    expect(result.current.data?.items.map((e) => e.id)).toEqual(all.slice(PAGE_SIZE, 2 * PAGE_SIZE).map((e) => e.id))

    const { result: filtered } = renderHook(
      () => useExpenses({ userId: 100000002, category: 'food', from: '2026-10-01', to: '2026-10-16' }),
      { wrapper },
    )
    await waitFor(() => expect(filtered.current.isSuccess).toBe(true))
    for (const e of filtered.current.data?.items ?? []) {
      expect(e).toMatchObject({ user_id: 100000002, category: 'food' })
      expect(e.spent_at >= '2026-10-01' && e.spent_at <= '2026-10-16').toBe(true)
    }
  })

  it('summarizes the whole filter, not one page', async () => {
    const { db, wrapper } = setup()
    const { result } = renderHook(() => useSummary({}), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const total = Math.round(db.expenses.reduce((s, e) => s + e.amount, 0) * 100) / 100
    expect(result.current.data?.total).toBe(total)
    expect(result.current.data?.count).toBe(db.expenses.length)
    const totals = result.current.data?.by_category.map((c) => c.total) ?? []
    expect(totals).toEqual([...totals].sort((a, b) => b - a))
  })

  it('loads more until everything is shown', async () => {
    const { db, wrapper } = setup()
    const { result } = renderHook(() => useExpensesInfinite({}), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    while (result.current.hasNextPage) {
      await act(() => result.current.fetchNextPage())
    }
    const shown = result.current.data?.pages.flatMap((p) => p.items) ?? []
    expect(shown).toHaveLength(db.expenses.length)
    expect(new Set(shown.map((e) => e.id)).size).toBe(db.expenses.length)
  })

  it('turns a server error into ApiError with field messages', async () => {
    const { wrapper } = setup()
    const { result } = renderHook(() => useExpenses({ from: '2026-10-16', to: '2026-10-01' }), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(ApiError)
    expect(result.current.error).toMatchObject({
      status: 400,
      code: 'bad_request',
      fields: { from: expect.any(String) },
    })
  })

  it('reports a network failure', async () => {
    const { wrapper } = setup()
    server.use(http.get('*/api/users', () => HttpResponse.error()))
    const { result } = renderHook(() => useUsers(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ status: 0, code: 'network' })
    expect(result.current.error?.message).toMatch(/сервер/)
  })
})

describe('mutations', () => {
  it('creates an expense and refreshes lists and totals', async () => {
    const { db, wrapper } = setup()
    const { result } = renderHook(
      () => ({ list: useExpenses({}), summary: useSummary({}), create: useCreateExpense() }),
      { wrapper },
    )
    await waitFor(() => expect(result.current.summary.isSuccess && result.current.list.isSuccess).toBe(true))
    const before = result.current.summary.data?.count ?? 0

    let created: Awaited<ReturnType<typeof result.current.create.mutateAsync>> | undefined
    await act(async () => {
      created = await result.current.create.mutateAsync({
        user_id: 100000002,
        amount: 349.9,
        category: 'health',
        spent_at: '2026-10-16',
        comment: '  аптека ',
      })
    })
    expect(created).toMatchObject({ amount: 349.9, source: 'web', comment: 'аптека', user_name: 'Мадина' })
    await waitFor(() => expect(result.current.summary.data?.count).toBe(before + 1))
    expect(result.current.list.data?.total).toBe(db.expenses.length)
  })

  it('exposes validation errors per field', async () => {
    const { wrapper } = setup()
    const { result } = renderHook(() => useCreateExpense(), { wrapper })
    await act(async () => {
      await result.current
        .mutateAsync({ user_id: 42, amount: 0, category: 'food', spent_at: '2026-10-16' })
        .catch(() => {})
    })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({
      status: 422,
      code: 'validation_error',
      fields: { user_id: expect.any(String), amount: expect.any(String) },
    })
  })

  it('updates and deletes', async () => {
    const { db, wrapper } = setup()
    const target = db.expenses[0]
    if (!target) throw new Error('no mock data')
    const { result } = renderHook(() => ({ update: useUpdateExpense(), remove: useDeleteExpense() }), { wrapper })

    await act(async () => {
      await result.current.update.mutateAsync({
        id: target.id,
        body: { user_id: 100000001, amount: 450, category: 'food', spent_at: target.spent_at, comment: null },
      })
    })
    expect(db.expenses.find((e) => e.id === target.id)).toMatchObject({ amount: 450, comment: null })

    await act(async () => {
      await result.current.remove.mutateAsync({ id: target.id })
    })
    expect(db.expenses.some((e) => e.id === target.id)).toBe(false)

    await act(async () => {
      await result.current.remove.mutateAsync({ id: target.id }).catch(() => {})
    })
    await waitFor(() => expect(result.current.remove.error).toMatchObject({ status: 404, code: 'not_found' }))
  })
})
