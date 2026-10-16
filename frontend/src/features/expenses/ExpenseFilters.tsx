import { CalendarDays, RotateCcw } from 'lucide-react'
import { useState } from 'react'

import { useUsers } from '@/api/hooks'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { CATEGORIES, CATEGORY_CODES, isCategoryCode } from '@/shared/categories'
import { initials } from '@/shared/people'
import { formatPeriod, matchPreset, PERIOD_PRESETS, presetPeriod, type PeriodPreset } from '@/shared/dates'

import { PeriodCalendar } from './PeriodCalendar'
import { defaultPeriod, hasDefaultFilters, type ExpensesView, type ViewPatch } from './urlState'

const ALL = 'all'

type Props = { view: ExpensesView; onChange: (patch: ViewPatch) => void }

/** Filter bar (≥ md): member, period with quick presets, category, reset. */
export function ExpenseFilters({ view, onChange }: Props) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <MemberSelect view={view} onChange={onChange} />
      <PeriodPicker view={view} onChange={onChange} />
      <PeriodPresets view={view} onChange={onChange} />
      <CategorySelect view={view} onChange={onChange} />
      {!hasDefaultFilters(view) && (
        <Button
          variant="ghost"
          onClick={() => onChange({ userId: undefined, category: undefined, ...defaultPeriod() })}
        >
          <RotateCcw />
          Сбросить
        </Button>
      )}
    </div>
  )
}

function MemberSelect({ view, onChange }: Props) {
  const users = useUsers()
  return (
    <Select
      value={view.userId === undefined ? ALL : String(view.userId)}
      onValueChange={(v) => onChange({ userId: v === ALL ? undefined : Number(v) })}
    >
      <SelectTrigger className="w-[200px]" aria-label="Член семьи">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ALL}>Все члены семьи</SelectItem>
        {users.data?.map((u) => (
          <SelectItem key={u.id} value={String(u.id)}>
            <Avatar className="size-5">
              <AvatarFallback className="bg-secondary text-[10px]">{initials(u.name)}</AvatarFallback>
            </Avatar>
            <span className="truncate">{u.name}</span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

function PeriodPicker({ view, onChange }: Props) {
  const [open, setOpen] = useState(false)
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" aria-label={`Период: ${formatPeriod(view)}`}>
          <CalendarDays />
          {formatPeriod(view)}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <PeriodCalendar
          period={view}
          numberOfMonths={2}
          onPick={(p) => {
            onChange(p)
            setOpen(false)
          }}
        />
      </PopoverContent>
    </Popover>
  )
}

function PeriodPresets({ view, onChange }: Props) {
  const active = matchPreset(view) ?? ''
  return (
    <ToggleGroup
      type="single"
      size="sm"
      spacing={0}
      value={active}
      onValueChange={(v) => v && onChange(presetPeriod(v as PeriodPreset))}
      className="rounded-lg bg-muted p-[3px]"
      aria-label="Быстрый выбор периода"
    >
      {PERIOD_PRESETS.map((p) => (
        <ToggleGroupItem key={p.id} value={p.id} className="px-3 data-[state=on]:bg-card data-[state=on]:shadow-sm">
          {p.label}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  )
}

function CategorySelect({ view, onChange }: Props) {
  return (
    <Select
      value={view.category ?? ALL}
      onValueChange={(v) => onChange({ category: isCategoryCode(v) ? v : undefined })}
    >
      <SelectTrigger className="w-[200px]" aria-label="Категория">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ALL}>Все категории</SelectItem>
        {CATEGORY_CODES.map((code) => (
          <SelectItem key={code} value={code}>
            <span className={`size-2.5 rounded-full ${CATEGORIES[code].fill}`} aria-hidden />
            {CATEGORIES[code].label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
