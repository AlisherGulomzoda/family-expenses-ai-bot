import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'

import type { MockDb } from '@/mocks/db'
import { resetMockApi, server, TEST_NOW } from '@/mocks/node'
import { renderApp } from '@/test/renderApp'

let db: MockDb

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(TEST_NOW)
  db = resetMockApi()
})
afterEach(() => vi.useRealTimers())

const thisMonth = () => db.expenses.filter((e) => e.spent_at >= '2026-10-01' && e.spent_at <= '2026-10-31')

/** Clicks 🗑 in the first table row and returns the confirmation and the expense id. */
async function deleteFirstRow(user: ReturnType<typeof userEvent.setup>) {
  const [bin] = await screen.findAllByRole('button', { name: /^Удалить: / })
  if (!bin) throw new Error('no rows')
  const countBefore = db.expenses.length
  await user.click(bin)
  const dialog = await screen.findByRole('alertdialog', { name: 'Удалить трату?' })
  return { dialog, countBefore }
}

describe('deleting from the table', () => {
  it('shows what will be deleted and does nothing on cancel', async () => {
    const user = userEvent.setup()
    renderApp()
    const { dialog, countBefore } = await deleteFirstRow(user)

    expect(within(dialog).getByText('Это действие нельзя отменить.')).toBeInTheDocument()
    const card = within(dialog).getByTestId('expense-to-delete')
    expect(card).toHaveTextContent(/₽/)
    expect(card).toHaveTextContent(/\d{2}\.\d{2}\.2026/)
    // Enter right after opening must not delete: focus starts on «Отмена».
    expect(within(dialog).getByRole('button', { name: 'Отмена' })).toHaveFocus()

    await user.click(within(dialog).getByRole('button', { name: 'Отмена' }))
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
    expect(db.expenses).toHaveLength(countBefore)
  })

  it('deletes after confirmation and says so', async () => {
    const user = userEvent.setup()
    renderApp()
    const { dialog, countBefore } = await deleteFirstRow(user)
    await user.click(within(dialog).getByRole('button', { name: 'Удалить' }))

    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
    expect(db.expenses).toHaveLength(countBefore - 1)
    expect(await screen.findByText('Трата удалена')).toBeInTheDocument()
  })

  it('keeps the dialog open with a retry when deleting fails', async () => {
    server.use(
      http.delete(
        '*/api/expenses/:id',
        () => HttpResponse.json({ error: { code: 'internal', message: 'Сбой базы' } }, { status: 500 }),
        {
          once: true,
        },
      ),
    )
    const user = userEvent.setup()
    renderApp()
    const { dialog, countBefore } = await deleteFirstRow(user)
    await user.click(within(dialog).getByRole('button', { name: 'Удалить' }))

    expect(await within(dialog).findByRole('alert')).toHaveTextContent('Не удалось удалить: Сбой базы')
    expect(db.expenses).toHaveLength(countBefore)
    await user.click(within(dialog).getByRole('button', { name: 'Повторить' }))
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
    expect(db.expenses).toHaveLength(countBefore - 1)
  })

  it('treats an already deleted expense as deleted', async () => {
    server.use(
      http.delete('*/api/expenses/:id', () =>
        HttpResponse.json({ error: { code: 'not_found', message: 'Трата не найдена' } }, { status: 404 }),
      ),
    )
    const user = userEvent.setup()
    renderApp()
    const { dialog } = await deleteFirstRow(user)
    await user.click(within(dialog).getByRole('button', { name: 'Удалить' }))
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
    expect(await screen.findByText('Трата уже удалена')).toBeInTheDocument()
  })

  it('goes back a page when the last expense on a page is deleted', async () => {
    // Exactly 11 expenses this month: page 2 holds one.
    const keep = new Set(
      thisMonth()
        .slice(0, 11)
        .map((e) => e.id),
    )
    db.expenses = db.expenses.filter((e) => keep.has(e.id))
    const user = userEvent.setup()
    renderApp('?page=2')
    await waitFor(() => expect(screen.getByText('Показано 11–11 из 11')).toBeInTheDocument())

    const { dialog } = await deleteFirstRow(user)
    await user.click(within(dialog).getByRole('button', { name: 'Удалить' }))
    await waitFor(() => expect(window.location.search).toBe(''))
    expect(await screen.findByText('Показано 1–10 из 10')).toBeInTheDocument()
  })

  it('fixes a link to a page past the end', async () => {
    renderApp('?page=99')
    await waitFor(() => expect(window.location.search).toBe('?page=2'))
  })
})

describe('deleting from the edit dialog', () => {
  it('returns to editing on cancel and closes both after deleting', async () => {
    const user = userEvent.setup()
    renderApp()
    const [pencil] = await screen.findAllByRole('button', { name: /^Изменить: / })
    if (!pencil) throw new Error('no rows')
    await user.click(pencil)
    const edit = await screen.findByRole('dialog', { name: 'Редактирование траты' })
    const countBefore = db.expenses.length

    await user.click(within(edit).getByRole('button', { name: 'Удалить' }))
    const confirm = await screen.findByRole('alertdialog', { name: 'Удалить трату?' })
    await user.click(within(confirm).getByRole('button', { name: 'Отмена' }))
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
    expect(screen.getByRole('dialog', { name: 'Редактирование траты' })).toBeInTheDocument()

    await user.click(within(edit).getByRole('button', { name: 'Удалить' }))
    const again = await screen.findByRole('alertdialog', { name: 'Удалить трату?' })
    await user.click(within(again).getByRole('button', { name: 'Удалить' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(db.expenses).toHaveLength(countBefore - 1)
  })
})
