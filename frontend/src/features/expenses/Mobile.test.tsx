import { act, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import type { MockDb } from '@/mocks/db'
import { resetMockApi, TEST_NOW } from '@/mocks/node'
import { renderApp } from '@/test/renderApp'
import { PHONE, setViewport } from '@/test/viewport'

import { countFilters } from './urlState'

let db: MockDb

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(TEST_NOW)
  db = resetMockApi()
  setViewport(PHONE)
})
afterEach(() => vi.useRealTimers())

const params = () => Object.fromEntries(new URLSearchParams(window.location.search))
const thisMonth = () => db.expenses.filter((e) => e.spent_at >= '2026-10-01' && e.spent_at <= '2026-10-31')
const cards = () =>
  within(screen.getByRole('region', { name: 'Траты' })).queryAllByRole('button', { name: /^Действия: / })

describe('phone layout', () => {
  it('shows cards, the filter button and a floating «+» instead of the table', async () => {
    renderApp()
    await waitFor(() => expect(cards()).toHaveLength(10))
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Фильтры' })).toBeInTheDocument()
    // Only one «Добавить трату»: the floating button.
    expect(screen.getAllByRole('button', { name: 'Добавить трату' })).toHaveLength(1)
  })

  it('loads ten more with «Показать ещё»', async () => {
    const user = userEvent.setup()
    renderApp()
    const total = thisMonth().length
    const more = await screen.findByRole('button', { name: `Показать ещё ${Math.min(10, total - 10)}` })
    await user.click(more)
    await waitFor(() => expect(cards()).toHaveLength(Math.min(20, total)))
    // The button stays only while there is more to load.
    expect(screen.queryByRole('button', { name: /Показать ещё/ }) !== null).toBe(total > 20)
  })

  it('edits from the card menu in a full-screen form with the amount focused', async () => {
    const user = userEvent.setup()
    renderApp()
    await waitFor(() => expect(cards()).toHaveLength(10))
    await user.click(cards()[0] as HTMLElement)
    await user.click(await screen.findByRole('menuitem', { name: 'Изменить' }))
    const dialog = await screen.findByRole('dialog', { name: 'Редактирование траты' })
    await waitFor(() => expect(within(dialog).getByLabelText('Сумма, ₽')).toHaveFocus())
    expect(dialog).toHaveClass('h-dvh')
  })

  it('deletes from the card menu after confirmation', async () => {
    const user = userEvent.setup()
    renderApp()
    await waitFor(() => expect(cards()).toHaveLength(10))
    const before = db.expenses.length
    await user.click(cards()[0] as HTMLElement)
    await user.click(await screen.findByRole('menuitem', { name: 'Удалить' }))
    const confirm = await screen.findByRole('alertdialog', { name: 'Удалить трату?' })
    await user.click(within(confirm).getByRole('button', { name: 'Удалить' }))
    await waitFor(() => expect(db.expenses).toHaveLength(before - 1))
  })
})

describe('phone filters', () => {
  it('applies the drawer only on «Применить»', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(screen.getByRole('button', { name: 'Фильтры' }))
    let drawer = await screen.findByRole('dialog', { name: 'Фильтры' })
    await user.click(await within(drawer).findByRole('radio', { name: /Мадина/ }))
    await user.click(within(drawer).getByRole('radio', { name: 'Еда' }))
    expect(params()).toEqual({}) // nothing applied yet

    await user.click(within(drawer).getByRole('button', { name: 'Применить' }))
    expect(params()).toEqual({ user_id: '100000002', category: 'food' })
    expect(await screen.findByRole('button', { name: 'Фильтры, выбрано: 2' })).toBeInTheDocument()

    // «Сбросить» in the drawer resets the draft; closing without «Применить» keeps the filters.
    await user.click(screen.getByRole('button', { name: 'Фильтры, выбрано: 2' }))
    drawer = await screen.findByRole('dialog', { name: 'Фильтры' })
    await user.click(within(drawer).getByRole('button', { name: 'Сбросить' }))
    await user.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Фильтры' })).not.toBeInTheDocument())
    expect(params()).toEqual({ user_id: '100000002', category: 'food' })
  })

  it('switches the period with the chips', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(screen.getByRole('radio', { name: 'Неделя' }))
    expect(params()).toEqual({ from: '2026-10-12', to: '2026-10-18' })
    expect(await screen.findByRole('button', { name: 'Фильтры, выбрано: 1' })).toBeInTheDocument()
  })

  it('counts member, period and category', () => {
    const month = { from: '2026-10-01', to: '2026-10-31' }
    expect(countFilters(month, month)).toBe(0)
    expect(countFilters({ ...month, userId: 1, category: 'food' }, month)).toBe(2)
    expect(countFilters({ from: '2026-09-01', to: '2026-09-30', userId: 1, category: 'food' }, month)).toBe(3)
  })
})

describe('phone summary', () => {
  it('shows the total, the count with the average and an expandable legend', async () => {
    const user = userEvent.setup()
    renderApp()
    const summary = await screen.findByRole('region', { name: 'Сводка' })
    const n = thisMonth().length
    await waitFor(() => expect(summary).toHaveTextContent(new RegExp(`${n} трат.? · в среднем`)))

    const toggle = within(summary).getByRole('button', { name: /^По категориям: \d+$/ })
    await user.click(toggle)
    expect(within(summary).getByRole('button', { name: 'Скрыть категории' })).toBeInTheDocument()
    expect(within(summary).getAllByRole('listitem').length).toBeGreaterThan(0)
  })
})

it('keeps the filters when the window crosses 768px', async () => {
  setViewport(1280)
  renderApp('?category=transport')
  expect(await screen.findByRole('table')).toBeInTheDocument()
  act(() => setViewport(PHONE))
  expect(await screen.findByRole('button', { name: 'Фильтры, выбрано: 1' })).toBeInTheDocument()
  expect(screen.queryByRole('table')).not.toBeInTheDocument()
  expect(params()).toEqual({ category: 'transport' })
})
