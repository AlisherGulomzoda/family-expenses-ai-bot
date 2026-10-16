import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { App } from '@/App'
import { Providers } from '@/app/Providers'
import { createQueryClient } from '@/app/queryClient'
import { resetMockApi, TEST_NOW } from '@/mocks/node'

function renderApp(search = '') {
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

beforeEach(() => {
  // Only Date is faked: "today" is 2026-10-16 while timers stay real.
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(TEST_NOW)
  resetMockApi()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('expenses page shell', () => {
  it('shows the header and the summary of this month', async () => {
    renderApp()
    expect(screen.getByRole('heading', { name: 'Семейные расходы' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Период: 1–31 окт 2026/ })).toBeInTheDocument()
    expect(await screen.findByText(/Сводка по всем \d+ тратам по фильтру/)).toBeInTheDocument()
  })

  it('switches the theme with the .dark class on <html>', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(await screen.findByRole('button', { name: 'Тёмная тема' }))
    expect(document.documentElement).toHaveClass('dark')
    await user.click(await screen.findByRole('button', { name: 'Светлая тема' }))
    expect(document.documentElement).not.toHaveClass('dark')
  })
})

describe('filters in the URL', () => {
  it('starts clean and offers no reset', () => {
    renderApp()
    expect(window.location.search).toBe('')
    expect(screen.queryByRole('button', { name: 'Сбросить' })).not.toBeInTheDocument()
    expect(screen.getByRole('radio', { name: 'Этот месяц' })).toHaveAttribute('data-state', 'on')
  })

  it('applies a quick period, then resets everything', async () => {
    const user = userEvent.setup()
    renderApp('?page=3')

    await user.click(screen.getByRole('radio', { name: 'Прошлый месяц' }))
    expect(params()).toEqual({ from: '2026-09-01', to: '2026-09-30' }) // page went back to 1
    expect(screen.getByRole('button', { name: /Период: 1–30 сен 2026/ })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Сбросить' }))
    expect(window.location.search).toBe('')
    expect(screen.queryByRole('button', { name: 'Сбросить' })).not.toBeInTheDocument()
  })

  it('filters by family member and category', async () => {
    const user = userEvent.setup()
    renderApp()

    await user.click(screen.getByRole('combobox', { name: 'Член семьи' }))
    await user.click(await screen.findByRole('option', { name: /Мадина/ }))
    expect(params()).toEqual({ user_id: '100000002' })

    await user.click(screen.getByRole('combobox', { name: 'Категория' }))
    const listbox = await screen.findByRole('listbox')
    expect(
      within(listbox)
        .getAllByRole('option')
        .map((o) => o.textContent),
    ).toEqual(['Все категории', 'Еда', 'Транспорт', 'Жильё', 'Развлечения', 'Здоровье', 'Покупки', 'Другое'])
    await user.click(within(listbox).getByRole('option', { name: 'Здоровье' }))
    expect(params()).toEqual({ user_id: '100000002', category: 'health' })
  })

  it('restores the filters from a shared link and ignores garbage', async () => {
    renderApp('?category=transport&from=2026-10-12&to=2026-10-18&sort=nope&page=abc')
    expect(screen.getByRole('combobox', { name: 'Категория' })).toHaveTextContent('Транспорт')
    expect(screen.getByRole('radio', { name: 'Неделя' })).toHaveAttribute('data-state', 'on')
    await waitFor(() => expect(screen.getByText(/Сводка по всем/)).toBeInTheDocument())
  })

  it('picks a custom period in the calendar', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(screen.getByRole('button', { name: /Период:/ }))
    // Day buttons are labelled by react-day-picker in Russian: "понедельник, 5 октября 2026 г.".
    // The calendar re-renders after the first click, so the day is looked up each time.
    const day = async (label: RegExp) => {
      const grid = (await screen.findAllByRole('grid'))[0]
      if (!grid) throw new Error('no calendar')
      return within(grid).getByRole('button', { name: label })
    }
    await user.click(await day(/\b5 октября/))
    expect(window.location.search).toBe('') // the first click only starts the range
    await user.click(await day(/\b9 октября/))
    expect(params()).toEqual({ from: '2026-10-05', to: '2026-10-09' })
    expect(screen.getByRole('button', { name: /Период: 5–9 окт 2026/ })).toBeInTheDocument()
    // A custom range matches no quick period.
    for (const radio of screen.getAllByRole('radio')) {
      expect(radio).toHaveAttribute('data-state', 'off')
    }
  })
})
