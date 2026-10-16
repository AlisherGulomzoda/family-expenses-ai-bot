import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import axe from 'axe-core'

import { resetMockApi, TEST_NOW } from '@/mocks/node'
import { renderApp } from '@/test/renderApp'
import { PHONE, setViewport } from '@/test/viewport'

// axe-core over the main states of the page. jsdom does no layout, so the rules
// that need rendering (colour contrast, target size) are left to the browser check.
async function violations() {
  const result = await axe.run(document.body, {
    rules: { 'color-contrast': { enabled: false }, 'target-size': { enabled: false } },
  })
  return result.violations.map((v) => `${v.id}: ${v.help} — ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`)
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(TEST_NOW)
  resetMockApi()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('accessibility (axe)', () => {
  it('desktop page with data', async () => {
    renderApp()
    await screen.findAllByRole('row', { name: /кофе/i })
    expect(await violations()).toEqual([])
  })

  it('create dialog with validation errors', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(await screen.findByRole('button', { name: 'Добавить трату' }))
    await screen.findByRole('dialog', { name: 'Новая трата' })
    await user.click(screen.getByRole('button', { name: 'Сохранить' }))
    await screen.findAllByRole('alert')
    expect(await violations()).toEqual([])
  })

  it('phone page and filter drawer', async () => {
    setViewport(PHONE)
    const user = userEvent.setup()
    renderApp()
    await screen.findAllByRole('button', { name: /^Действия: / })
    expect(await violations()).toEqual([])
    await user.click(screen.getByRole('button', { name: 'Фильтры' }))
    await screen.findByRole('dialog', { name: 'Фильтры' })
    expect(await violations()).toEqual([])
  })
})
