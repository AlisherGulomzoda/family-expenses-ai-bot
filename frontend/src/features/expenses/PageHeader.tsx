import { Plus } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { ThemeToggle } from '@/theme/ThemeToggle'

/** Title, theme switch and the main action (design-spec §4 «Шапка и фильтры»). */
export function PageHeader({ onAdd }: { onAdd?: () => void }) {
  return (
    <header className="flex items-center justify-between gap-3">
      <h1 className="text-xl font-semibold tracking-tight md:text-2xl">Семейные расходы</h1>
      <div className="flex items-center gap-2">
        <ThemeToggle />
        <Button onClick={onAdd}>
          <Plus />
          Добавить трату
        </Button>
      </div>
    </header>
  )
}
