import { CalendarDays, RotateCcw } from 'lucide-react'
import { useState } from 'react'
import { ru } from 'react-day-picker/locale'

import { useUsers } from '@/api/hooks'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { CATEGORIES, CATEGORY_CODES, isCategoryCode } from '@/shared/categories'
import { initials } from '@/shared/people'
import {
  formatPeriod,
  fromISODate,
  matchPreset,
  PERIOD_PRESETS,
  presetPeriod,
  toISODate,
  type PeriodPreset,
} from '@/shared/dates'

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

/**
 * Two clicks pick a period: the first starts a new range, the second ends it
 * (the same day twice is a one-day period). react-day-picker would instead
 * stretch the current range, so the clicks are interpreted here.
 */
function PeriodPicker({ view, onChange }: Props) {
  const [open, setOpen] = useState(false)
  const [start, setStart] = useState<Date>()
  const current = { from: fromISODate(view.from), to: fromISODate(view.to) }

  const pick = (day: Date) => {
    if (!start) {
      setStart(day)
      return
    }
    const [from, to] = start <= day ? [start, day] : [day, start]
    onChange({ from: toISODate(from), to: toISODate(to) })
    setStart(undefined)
    setOpen(false)
  }

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        setStart(undefined)
      }}
    >
      <PopoverTrigger asChild>
        <Button variant="outline" aria-label={`Период: ${formatPeriod(view)}`}>
          <CalendarDays />
          {formatPeriod(view)}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="range"
          numberOfMonths={2}
          locale={ru}
          defaultMonth={current.from}
          // `selected` only highlights; the clicks themselves are handled by pick().
          selected={start ? { from: start, to: start } : current}
          onDayClick={(day, modifiers) => !modifiers.disabled && pick(day)}
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
