import { Plus } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { ThemeToggle } from '@/theme/ThemeToggle'

/**
 * Title, theme switch and the main action (design-spec §4 «Шапка и фильтры»).
 * On phones the action is the floating «+» instead (see FloatingAddButton).
 */
export function PageHeader({ onAdd, compact }: { onAdd?: () => void; compact?: boolean }) {
  return (
    <header className="flex items-center justify-between gap-3">
      <h1 className="text-xl font-semibold tracking-tight md:text-2xl">Семейные расходы</h1>
      <div className="flex items-center gap-2">
        <ThemeToggle compact={compact} />
        {!compact && (
          <Button onClick={onAdd}>
            <Plus />
            Добавить трату
          </Button>
        )}
      </div>
    </header>
  )
}

/** Phone «+» in the bottom right corner. */
export function FloatingAddButton({ onAdd }: { onAdd: () => void }) {
  return (
    <Button
      size="icon"
      className="fixed right-4 bottom-6 z-40 size-14 rounded-full shadow-lg [&_svg:not([class*='size-'])]:size-6"
      aria-label="Добавить трату"
      onClick={onAdd}
    >
      <Plus />
    </Button>
  )
}
