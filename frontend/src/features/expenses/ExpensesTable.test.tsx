import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { delay, http, HttpResponse } from 'msw'

import type { Expense } from '@/api/client'
import { App } from '@/App'
import { Providers } from '@/app/Providers'
import { createQueryClient } from '@/app/queryClient'
import type { MockDb } from '@/mocks/db'
import { resetMockApi, server, TEST_NOW } from '@/mocks/node'

import { pageItems } from './paging'

let db: MockDb

function renderPage(search = '') {
  window.history.replaceState(null, '', `/${search}`)
  const client = createQueryClient()
  client.setDefaultOptions({ queries: { retry: false } })
  return render(
    <Providers queryClient={client}>
      <App />
    </Providers>,
  )
}

const params = () => Object.fromEntries(new URLSearchParams(window.location.search))
const bodyRows = () => within(screen.getByRole('table')).getAllByRole('row').slice(1)

/** This month's expenses as the API orders them by default. */
function expected(sort: (a: Expense, b: Expense) => number = byDateDesc) {
  return db.expenses.filter((e) => e.spent_at >= '2026-10-01' && e.spent_at <= '2026-10-31').sort(sort)
}
const byDateDesc = (a: Expense, b: Expense) =>
  a.spent_at < b.spent_at ? 1 : a.spent_at > b.spent_at ? -1 : b.id - a.id
const byAmountAsc = (a: Expense, b: Expense) => a.amount - b.amount || a.id - b.id

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(TEST_NOW)
  db = resetMockApi()
})
afterEach(() => vi.useRealTimers())

describe('expenses table', () => {
  it('shows the first page of this month, newest first', async () => {
    renderPage()
    await waitFor(() => expect(bodyRows()).toHaveLength(10))
    const want = expected()
    const first = want[0]
    if (!first) throw new Error('no mock data this month')
    const cells = within(bodyRows()[0] as HTMLElement).getAllByRole('cell')
    expect(cells[0]).toHaveTextContent(first.spent_at.split('-').reverse().join('.'))
    expect(cells[1]).toHaveTextContent(first.user_name)
    expect(screen.getByText(`Показано 1–10 из ${want.length}`)).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: /Дата/ })).toHaveAttribute('aria-sort', 'descending')
    expect(screen.getByRole('columnheader', { name: /Сумма/ })).toHaveAttribute('aria-sort', 'none')
  })

  it('sorts by amount on the server and toggles the direction', async () => {
    const user = userEvent.setup()
    renderPage('?page=2')
    await waitFor(() => expect(screen.getByText(/Показано 11–/)).toBeInTheDocument())

    await user.click(screen.getByRole('button', { name: /Сумма/ }))
    expect(params()).toEqual({ sort: 'amount' }) // page went back to 1, order desc is the default
    await user.click(screen.getByRole('button', { name: /Сумма/ }))
    expect(params()).toEqual({ sort: 'amount', order: 'asc' })
    expect(screen.getByRole('columnheader', { name: /Сумма/ })).toHaveAttribute('aria-sort', 'ascending')

    const cheapest = expected(byAmountAsc)[0]
    if (!cheapest) throw new Error('no data')
    await waitFor(() =>
      expect(within(bodyRows()[0] as HTMLElement).getAllByRole('cell')[3]).toHaveTextContent(
        new Intl.NumberFormat('ru-RU').format(Math.trunc(cheapest.amount)).replace(/\s/g, ' '),
      ),
    )
  })

  it('pages with real links', async () => {
    const user = userEvent.setup()
    renderPage()
    await waitFor(() => expect(bodyRows()).toHaveLength(10))
    const total = expected().length

    const page2 = screen.getByRole('link', { name: 'Страница 2' })
    expect(page2).toHaveAttribute('href', '?page=2')
    await user.click(page2)
    expect(params()).toEqual({ page: '2' })
    await waitFor(() => expect(screen.getByText(`Показано 11–${Math.min(20, total)} из ${total}`)).toBeInTheDocument())
    expect(screen.getByRole('link', { name: 'Страница 2' })).toHaveAttribute('aria-current', 'page')

    await user.click(screen.getByRole('link', { name: 'Предыдущая страница' }))
    expect(params()).toEqual({})
  })

  it('labels row actions with the expense', async () => {
    renderPage()
    await waitFor(() => expect(bodyRows()).toHaveLength(10))
    const first = expected()[0]
    if (!first) throw new Error('no data')
    const row = bodyRows()[0] as HTMLElement
    expect(
      within(row).getByRole('button', { name: new RegExp(`^Удалить: ${first.comment ?? ''}`) }),
    ).toBeInTheDocument()
    expect(within(row).getByRole('button', { name: /^Изменить: / })).toBeInTheDocument()
  })

  it('offers a reset when the filter finds nothing', async () => {
    const user = userEvent.setup()
    renderPage('?from=2020-01-01&to=2020-01-31')
    expect(
      await screen.findByText('За выбранный период трат нет', { selector: '[data-slot=empty-title]' }),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Сбросить фильтры' }))
    expect(window.location.search).toBe('')
    await waitFor(() => expect(bodyRows()).toHaveLength(10))
  })

  it('shows an error with a retry', async () => {
    const user = userEvent.setup()
    server.use(
      http.get(
        '*/api/expenses',
        () => HttpResponse.json({ error: { code: 'internal', message: 'x' } }, { status: 500 }),
        {
          once: true,
        },
      ),
    )
    renderPage()
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Не удалось загрузить траты')
    await user.click(within(alert).getByRole('button', { name: 'Повторить' }))
    await waitFor(() => expect(bodyRows()).toHaveLength(10))
  })

  it('shows skeleton rows with real headers while loading', async () => {
    server.use(
      http.get('*/api/expenses', async () => {
        await delay('infinite')
        return HttpResponse.json({})
      }),
    )
    renderPage()
    const table = await screen.findByRole('table')
    expect(table).toHaveAttribute('aria-busy', 'true')
    expect(within(table).getByRole('columnheader', { name: /Дата/ })).toBeInTheDocument()
    expect(bodyRows()).toHaveLength(6)
    expect(screen.queryByText(/Показано/)).not.toBeInTheDocument()
  })
})

describe('pageItems', () => {
  it.each([
    [1, 1, [1]],
    [3, 7, [1, 2, 3, 4, 5, 6, 7]],
    [1, 12, [1, 2, 3, 4, 'gap', 12]],
    [6, 12, [1, 'gap', 5, 6, 7, 'gap', 12]],
    [12, 12, [1, 'gap', 9, 10, 11, 12]],
    [4, 12, [1, 2, 3, 4, 5, 'gap', 12]],
  ])('page %i of %i', (current, pages, want) => {
    expect(pageItems(current, pages)).toEqual(want)
  })
})
