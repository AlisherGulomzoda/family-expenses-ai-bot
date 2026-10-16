import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { App } from '@/App'
import { Providers } from '@/app/Providers'

function renderApp() {
  return render(
    <Providers>
      <App />
    </Providers>,
  )
}

describe('App shell', () => {
  it('renders the title and all seven categories', () => {
    renderApp()
    expect(screen.getByRole('heading', { name: 'Семейные расходы' })).toBeInTheDocument()
    for (const label of ['Еда', 'Транспорт', 'Жильё', 'Развлечения', 'Здоровье', 'Покупки', 'Другое']) {
      expect(screen.getByText(label)).toBeInTheDocument()
    }
  })

  it('switches between light and dark theme with the .dark class on <html>', async () => {
    const user = userEvent.setup()
    renderApp()

    await user.click(await screen.findByRole('button', { name: 'Тёмная тема' }))
    expect(document.documentElement).toHaveClass('dark')

    await user.click(await screen.findByRole('button', { name: 'Светлая тема' }))
    expect(document.documentElement).not.toHaveClass('dark')
  })
})
