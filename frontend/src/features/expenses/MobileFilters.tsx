import { CalendarDays, SlidersHorizontal } from 'lucide-react'
import { useState } from 'react'

import { useUsers } from '@/api/hooks'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from '@/components/ui/drawer'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { cn } from '@/lib/utils'
import { CATEGORIES, CATEGORY_CODES, isCategoryCode, type CategoryCode } from '@/shared/categories'
import { formatPeriod, matchPreset, PERIOD_PRESETS, presetPeriod, type Period, type PeriodPreset } from '@/shared/dates'
import { initials } from '@/shared/people'

import { PeriodCalendar } from './PeriodCalendar'
import { countFilters, defaultPeriod, type ExpensesView, type ViewPatch } from './urlState'

type Props = { view: ExpensesView; onChange: (patch: ViewPatch) => void }

const ALL = 'all'

/** Filters the drawer holds until «Применить». */
type Draft = { userId?: number; category?: CategoryCode } & Period

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

function PresetChips({
  value,
  onChange,
  label,
}: {
  value: string
  onChange: (p: PeriodPreset) => void
  label: string
}) {
  return (
    <ToggleGroup
      type="single"
      variant="outline"
      spacing={2}
      value={value}
      onValueChange={(v) => v && onChange(v as PeriodPreset)}
      className="w-max"
      aria-label={label}
    >
      {PERIOD_PRESETS.map((p) => (
        <ToggleGroupItem
          key={p.id}
          value={p.id}
          className="h-11 rounded-full px-4 whitespace-nowrap data-[state=on]:bg-foreground data-[state=on]:text-background"
        >
          {p.label}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  )
}

function FiltersDrawer({
  open,
  onOpenChange,
  view,
  onApply,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  view: ExpensesView
  onApply: (patch: ViewPatch) => void
}) {
  const users = useUsers()
  const fromView = (): Draft => ({ userId: view.userId, category: view.category, from: view.from, to: view.to })
  const [draft, setDraft] = useState<Draft>(fromView)
  const [pickDates, setPickDates] = useState(false)
  const set = (patch: Partial<Draft>) => setDraft((d) => ({ ...d, ...patch }))

  return (
    <Drawer
      open={open}
      onOpenChange={(next) => {
        // Each opening starts from what is applied now.
        if (next) setDraft(fromView())
        setPickDates(false)
        onOpenChange(next)
      }}
    >
      <DrawerContent className="max-h-[90dvh] rounded-t-xl">
        <DrawerHeader className="flex-row items-center justify-between text-left">
          <div>
            <DrawerTitle>Фильтры</DrawerTitle>
            <DrawerDescription className="sr-only">Член семьи, период и категория</DrawerDescription>
          </div>
          <Button
            variant="link"
            className="h-11 px-0"
            onClick={() => setDraft({ ...defaultPeriod(), userId: undefined, category: undefined })}
          >
            Сбросить
          </Button>
        </DrawerHeader>

        <div className="flex flex-col gap-5 overflow-y-auto px-4 pb-2">
          <section className="flex flex-col gap-2">
            <h3 id="drawer-member" className="text-sm font-medium">
              Член семьи
            </h3>
            <ToggleGroup
              type="single"
              variant="outline"
              spacing={2}
              value={draft.userId === undefined ? ALL : String(draft.userId)}
              onValueChange={(v) => v && set({ userId: v === ALL ? undefined : Number(v) })}
              className="flex w-full flex-wrap justify-start"
              aria-labelledby="drawer-member"
            >
              <ToggleGroupItem value={ALL} className={chip}>
                Все
              </ToggleGroupItem>
              {users.data?.map((u) => (
                <ToggleGroupItem key={u.id} value={String(u.id)} className={chip}>
                  <Avatar className="size-6">
                    <AvatarFallback className="bg-secondary text-[10px]">{initials(u.name)}</AvatarFallback>
                  </Avatar>
                  {u.name}
                </ToggleGroupItem>
              ))}
            </ToggleGroup>
          </section>

          <section className="flex flex-col gap-2">
            <h3 className="text-sm font-medium">Период</h3>
            <div className="flex flex-wrap gap-2">
              <PresetChips value={matchPreset(draft) ?? ''} onChange={(p) => set(presetPeriod(p))} label="Период" />
            </div>
            <Button variant="outline" className="h-11 justify-between" onClick={() => setPickDates((v) => !v)}>
              <span>{matchPreset(draft) ? 'Выбрать даты' : formatPeriod(draft)}</span>
              <CalendarDays className="text-muted-foreground" />
            </Button>
            {pickDates && (
              <PeriodCalendar
                period={draft}
                className="mx-auto rounded-lg border"
                onPick={(p) => {
                  set(p)
                  setPickDates(false)
                }}
              />
            )}
          </section>

          <section className="flex flex-col gap-2">
            <h3 id="drawer-category" className="text-sm font-medium">
              Категория
            </h3>
            <ToggleGroup
              type="single"
              variant="outline"
              spacing={2}
              value={draft.category ?? ALL}
              onValueChange={(v) => v && set({ category: isCategoryCode(v) ? v : undefined })}
              className="flex w-full flex-wrap justify-start"
              aria-labelledby="drawer-category"
            >
              <ToggleGroupItem value={ALL} className={chip}>
                Все
              </ToggleGroupItem>
              {CATEGORY_CODES.map((code) => (
                <ToggleGroupItem key={code} value={code} className={chip}>
                  <span className={cn('size-2.5 rounded-full', CATEGORIES[code].fill)} aria-hidden />
                  {CATEGORIES[code].label}
                </ToggleGroupItem>
              ))}
            </ToggleGroup>
          </section>
        </div>

        <DrawerFooter>
          <Button
            size="lg"
            className="h-11 w-full"
            onClick={() => {
              onApply({ userId: draft.userId, category: draft.category, from: draft.from, to: draft.to })
              onOpenChange(false)
            }}
          >
            Применить
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  )
}

const chip = 'h-11 rounded-full px-3 data-[state=on]:bg-foreground data-[state=on]:text-background'
