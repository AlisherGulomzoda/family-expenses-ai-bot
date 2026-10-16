import { SlidersHorizontal } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { matchPreset, presetPeriod } from '@/shared/dates'

import { FiltersDrawer } from './lazy'
import { PresetChips } from './PresetChips'
import { countFilters, type ExpensesView, type ViewPatch } from './urlState'

type Props = { view: ExpensesView; onChange: (patch: ViewPatch) => void }

/** Phone filters (< 768px): «Фильтры (N)» opening a drawer, and period chips (design: MobileMain, MobileFilters). */
export function MobileFilters({ view, onChange }: Props) {
  const [open, setOpen] = useState(false)
  const count = countFilters(view)
  return (
    <div className="flex items-center gap-2">
      <Button
        variant="outline"
        className="h-11 shrink-0 rounded-full"
        onClick={() => setOpen(true)}
        aria-label={count > 0 ? `Фильтры, выбрано: ${count}` : 'Фильтры'}
      >
        <SlidersHorizontal />
        Фильтры
        {count > 0 && <Badge className="ml-0.5 tabular-nums">{count}</Badge>}
      </Button>
      <div className="-mr-4 [scrollbar-width:none] overflow-x-auto pr-4">
        <PresetChips
          value={matchPreset(view) ?? ''}
          onChange={(preset) => onChange(presetPeriod(preset))}
          label="Быстрый выбор периода"
        />
      </div>
      <FiltersDrawer open={open} onOpenChange={setOpen} view={view} onApply={onChange} />
    </div>
  )
}
