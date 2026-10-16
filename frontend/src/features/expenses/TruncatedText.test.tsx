import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { TooltipProvider } from '@/components/ui/tooltip'

import { TruncatedText } from './TruncatedText'

// jsdom has no layout: pretend the text is (or is not) wider than its box.
function fakeWidths(scrollWidth: number, clientWidth: number) {
  vi.spyOn(HTMLElement.prototype, 'scrollWidth', 'get').mockReturnValue(scrollWidth)
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(clientWidth)
}

const long = 'Продукты на неделю: овощи, фрукты, крупы, молочка и ещё много всего для праздничного ужина'

function renderText() {
  return render(
    <TooltipProvider delayDuration={0}>
      <TruncatedText text={long} />
    </TooltipProvider>,
  )
}

afterEach(() => vi.restoreAllMocks())

describe('TruncatedText', () => {
  it('shows the full text on focus when it is cut', async () => {
    fakeWidths(800, 200)
    renderText()
    await userEvent.setup().tab()
    expect(screen.getByRole('button', { name: long })).toHaveFocus()
    expect(await screen.findByRole('tooltip')).toHaveTextContent(long)
  })

  it('shows no tooltip when the text fits', async () => {
    fakeWidths(200, 200)
    renderText()
    await userEvent.setup().tab()
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })
})
