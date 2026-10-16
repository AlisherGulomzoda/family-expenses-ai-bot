import { QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import { delay, http, HttpResponse } from 'msw'

import type { ExpenseFilters } from '@/api/hooks'
import { createQueryClient } from '@/app/queryClient'
import { resetMockApi, server } from '@/mocks/node'
import { plural } from '@/shared/plural'

import { SummarySection } from './SummarySection'
import { average, share } from './stats'

const NBSP = ' '

function renderSummary(filters: ExpenseFilters = {}) {
  const client = createQueryClient()
  client.setDefaultOptions({ queries: { retry: false } })
  return render(
    <QueryClientProvider client={client}>
      <SummarySection filters={filters} />
    </QueryClientProvider>,
  )
}

const summaryJson = (body: Record<string, unknown>) =>
  server.use(http.get('*/api/expenses/summary', () => HttpResponse.json(body)))

describe('SummarySection', () => {
  beforeEach(() => resetMockApi())

  it('shows totals over the whole filter and the categories by amount', async () => {
    summaryJson({
      total: 48720,
      count: 37,
      by_category: [
        { category: 'food', total: 16400, count: 15 },
        { category: 'housing', total: 12000, count: 2 },
        { category: 'other', total: 820.5, count: 1 },
      ],
    })
    renderSummary()

    expect(await screen.findByText('48 720 ₽')).toBeInTheDocument()
    expect(screen.getByText('37')).toBeInTheDocument()
    expect(screen.getByText('1 316,76 ₽')).toBeInTheDocument() // 48720 / 37 (Testing Library normalizes NBSP to spaces)
    expect(screen.getByText('Сводка по всем 37 тратам по фильтру')).toBeInTheDocument()

    const rows = within(screen.getByText('По категориям').closest('[data-slot=card]') as HTMLElement).getAllByRole(
      'listitem',
    )
    expect(rows.map((r) => r.textContent)).toEqual([
      `Еда16${NBSP}400${NBSP}₽34%`,
      `Жильё12${NBSP}000${NBSP}₽25%`,
      `Другое820,50${NBSP}₽2%`,
    ])
    // Bars are relative to the largest category.
    expect(screen.getByRole('progressbar', { name: /Еда/ })).toHaveAttribute('aria-valuenow', '100')
    expect(screen.getByRole('progressbar', { name: /Другое/ })).toHaveAttribute(
      'aria-valuenow',
      String((820.5 / 16400) * 100),
    )
  })

  it('uses real data from the API', async () => {
    const db = resetMockApi()
    renderSummary({ userId: 100000002 })
    const own = db.expenses.filter((e) => e.user_id === 100000002)
    expect(
      await screen.findByText(
        new RegExp(`Сводка по всем ${own.length} ${plural(own.length, 'трате', 'тратам', 'тратам')}`),
      ),
    ).toBeInTheDocument()
  })

  it('shows zeros and an empty breakdown when nothing matches', async () => {
    renderSummary({ from: '2020-01-01', to: '2020-01-31' })
    expect(await screen.findByText('За выбранный период трат нет')).toBeInTheDocument()
    expect(screen.getAllByText('0 ₽')).toHaveLength(2) // total and average
    expect(screen.getByText('0')).toBeInTheDocument()
  })

  it('shows dashes when the summary fails', async () => {
    server.use(
      http.get('*/api/expenses/summary', () =>
        HttpResponse.json({ error: { code: 'internal', message: 'x' } }, { status: 500 }),
      ),
    )
    renderSummary()
    expect(await screen.findByText('Не удалось загрузить сводку')).toBeInTheDocument()
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(3)
  })

  it('shows skeletons while loading', async () => {
    server.use(
      http.get('*/api/expenses/summary', async () => {
        await delay('infinite')
        return HttpResponse.json({})
      }),
    )
    renderSummary()
    expect(screen.getByRole('region', { name: 'Сводка' })).toHaveAttribute('aria-busy', 'true')
    expect(screen.getByText('Считаем сводку…')).toBeInTheDocument()
  })
})

describe('stats', () => {
  it('computes the average and shares', () => {
    expect(average({ total: 48720, count: 37 })).toBe(1316.76)
    expect(average({ total: 0, count: 0 })).toBe(0)
    expect(share(16400, 48720)).toBe(34)
    expect(share(5, 0)).toBe(0)
  })

  it('declines Russian words', () => {
    expect([1, 2, 5, 11, 21, 22, 25, 111, 0].map((n) => plural(n, 'трата', 'траты', 'трат'))).toEqual([
      'трата',
      'траты',
      'трат',
      'трат',
      'трата',
      'траты',
      'трат',
      'трат',
      'трат',
    ])
  })
})
