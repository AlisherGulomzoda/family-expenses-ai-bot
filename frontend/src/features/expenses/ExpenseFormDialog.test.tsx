import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { delay, http, HttpResponse } from 'msw'

import type { MockDb } from '@/mocks/db'
import { resetMockApi, server, TEST_NOW } from '@/mocks/node'
import { renderApp } from '@/test/renderApp'

import { defaultMember, formSchema, toBody, type FormValues } from './expenseForm'

let db: MockDb

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(TEST_NOW)
  db = resetMockApi()
})
afterEach(() => vi.useRealTimers())

/** Records POST/PUT bodies while still answering with the mock API. */
function spyBodies(method: 'post' | 'put') {
  const bodies: unknown[] = []
  server.events.on('request:start', ({ request }) => {
    if (request.method === method.toUpperCase() && new URL(request.url).pathname.startsWith('/api/expenses')) {
      void request
        .clone()
        .json()
        .then((b) => bodies.push(b))
    }
  })
  return bodies
}

async function openCreate() {
  const user = userEvent.setup()
  renderApp()
  await user.click(screen.getByRole('button', { name: 'Добавить трату' }))
  const dialog = await screen.findByRole('dialog', { name: 'Новая трата' })
  // The first family member is selected once the list arrives.
  await waitFor(() => expect(within(dialog).getByRole('combobox', { name: 'Член семьи' })).toHaveTextContent('Алишер'))
  return { user, dialog }
}

describe('new expense', () => {
  it('validates before sending anything', async () => {
    const posts = spyBodies('post')
    const { user, dialog } = await openCreate()
    expect(within(dialog).getByRole('button', { name: 'Дата' })).toHaveTextContent('Сегодня, 16.10.2026')

    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(await within(dialog).findByText('Укажите сумму')).toBeInTheDocument()
    expect(within(dialog).getByText('Выберите категорию')).toBeInTheDocument()
    expect(within(dialog).getByLabelText('Сумма, ₽')).toHaveAttribute('aria-invalid', 'true')

    await user.type(within(dialog).getByLabelText('Сумма, ₽'), '12,345')
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(await within(dialog).findByText(/не больше двух знаков/)).toBeInTheDocument()
    expect(posts).toHaveLength(0)
  })

  it('creates an expense and updates the page', async () => {
    const posts = spyBodies('post')
    const before = db.expenses.length
    const { user, dialog } = await openCreate()

    await user.click(within(dialog).getByRole('combobox', { name: 'Член семьи' }))
    await user.click(await screen.findByRole('option', { name: /Мадина/ }))
    await user.type(within(dialog).getByLabelText('Сумма, ₽'), '349,90')
    await user.click(within(dialog).getByRole('radio', { name: 'Здоровье' }))
    await user.type(within(dialog).getByLabelText(/Комментарий/), '  аптека ')
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(posts).toEqual([
      { user_id: 100000002, amount: 349.9, category: 'health', spent_at: '2026-10-16', comment: 'аптека' },
    ])
    expect(db.expenses).toHaveLength(before + 1)
    expect(await screen.findByText('Трата добавлена')).toBeInTheDocument()
    expect(screen.getByText('Аптека · 349,90 ₽ · Мадина')).toBeInTheDocument()

    // The member chosen last time is the default next time.
    await user.click(screen.getByRole('button', { name: 'Добавить трату' }))
    const again = await screen.findByRole('dialog', { name: 'Новая трата' })
    await waitFor(() => expect(within(again).getByRole('combobox', { name: 'Член семьи' })).toHaveTextContent('Мадина'))
  })

  it('takes the default member from the filter', async () => {
    const user = userEvent.setup()
    renderApp('?user_id=100000003')
    await user.click(screen.getByRole('button', { name: 'Добавить трату' }))
    const dialog = await screen.findByRole('dialog', { name: 'Новая трата' })
    await waitFor(() => expect(within(dialog).getByRole('combobox', { name: 'Член семьи' })).toHaveTextContent('Тимур'))
  })

  it('shows server validation errors under the fields', async () => {
    server.use(
      http.post('*/api/expenses', () =>
        HttpResponse.json(
          { error: { code: 'validation_error', message: 'Проверьте', fields: { amount: 'Слишком много для кофе' } } },
          { status: 422 },
        ),
      ),
    )
    const { user, dialog } = await openCreate()
    await user.type(within(dialog).getByLabelText('Сумма, ₽'), '350')
    await user.click(within(dialog).getByRole('radio', { name: 'Еда' }))
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(await within(dialog).findByText('Слишком много для кофе')).toBeInTheDocument()
    expect(within(dialog).getByLabelText('Сумма, ₽')).toHaveFocus()
  })

  it('offers a retry when saving fails', async () => {
    server.use(
      http.post(
        '*/api/expenses',
        () => HttpResponse.json({ error: { code: 'internal', message: 'Что-то пошло не так' } }, { status: 500 }),
        {
          once: true,
        },
      ),
    )
    const { user, dialog } = await openCreate()
    await user.type(within(dialog).getByLabelText('Сумма, ₽'), '350')
    await user.click(within(dialog).getByRole('radio', { name: 'Еда' }))
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))

    // The error and the retry are inside the dialog: a modal dialog blocks toasts.
    const alert = await within(dialog).findByRole('alert')
    expect(alert).toHaveTextContent('Не удалось сохранить')
    expect(alert).toHaveTextContent('Что-то пошло не так')
    expect(within(dialog).getByLabelText('Сумма, ₽')).toHaveValue('350') // nothing typed is lost
    await user.click(within(alert).getByRole('button', { name: 'Повторить' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(await screen.findByText('Трата добавлена')).toBeInTheDocument()
  })

  it('shows progress while saving', async () => {
    server.use(
      http.post('*/api/expenses', async () => {
        await delay('infinite')
        return HttpResponse.json({})
      }),
    )
    const { user, dialog } = await openCreate()
    await user.type(within(dialog).getByLabelText('Сумма, ₽'), '350')
    await user.click(within(dialog).getByRole('radio', { name: 'Еда' }))
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(await within(dialog).findByRole('button', { name: /Сохраняем…/ })).toBeDisabled()
  })
})

describe('editing', () => {
  it('opens the expense, saves changes with PUT and keeps the source', async () => {
    const puts = spyBodies('put')
    const user = userEvent.setup()
    renderApp()
    // Rows have action buttons once the data (not the skeleton) is shown.
    const [edit] = await screen.findAllByRole('button', { name: /^Изменить: / })
    if (!edit) throw new Error('no rows')
    await user.click(edit)

    const dialog = await screen.findByRole('dialog', { name: 'Редактирование траты' })
    const amountInput = within(dialog).getByLabelText('Сумма, ₽')
    expect(amountInput).not.toHaveValue('')
    // The description tells where the expense came from.
    expect(within(dialog).getByText(/Записано через Telegram-бота|Добавлено в веб-панели/)).toBeInTheDocument()

    await user.clear(amountInput)
    await user.type(amountInput, '450')
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(puts).toHaveLength(1)
    expect(puts[0]).toMatchObject({ amount: 450 })
    expect(await screen.findByText('Изменения сохранены')).toBeInTheDocument()
  })
})

describe('form rules', () => {
  const valid: FormValues = { userId: '1', amount: '350', category: 'food', spentAt: '2026-10-16', comment: '' }

  it('allows tomorrow but not later', () => {
    expect(formSchema.safeParse({ ...valid, spentAt: '2026-10-17' }).success).toBe(true)
    expect(formSchema.safeParse({ ...valid, spentAt: '2026-10-18' }).success).toBe(false)
  })

  it('rejects zero and too large amounts', () => {
    expect(formSchema.safeParse({ ...valid, amount: '0' }).success).toBe(false)
    expect(formSchema.safeParse({ ...valid, amount: '100000001' }).success).toBe(false)
  })

  it('builds the API body without float noise', () => {
    expect(toBody({ ...valid, amount: '0,30', comment: '  ' })).toEqual({
      user_id: 1,
      amount: 0.3,
      category: 'food',
      spent_at: '2026-10-16',
      comment: null,
    })
  })

  it('picks the default member: filter, then last chosen, then first', () => {
    const users = [
      { id: 1, name: 'А', username: null },
      { id: 2, name: 'Б', username: null },
    ]
    expect(defaultMember(users, 7)).toBe(7)
    expect(defaultMember(users)).toBe(1)
    localStorage.setItem('expenses.lastMemberId', '2')
    expect(defaultMember(users)).toBe(2)
    localStorage.setItem('expenses.lastMemberId', '99') // left the family
    expect(defaultMember(users)).toBe(1)
  })
})
