import type { Summary } from '@/api/client'
import { useSummary, type ExpenseFilters } from '@/api/hooks'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { CATEGORIES, type CategoryCode } from '@/shared/categories'
import { formatRub } from '@/shared/money'
import { plural } from '@/shared/plural'
import { useIsDesktop } from '@/shared/useMediaQuery'

import { MobileSummary } from './MobileSummary'
import { average, share } from './stats'

// Progress paints its only child (the indicator); the class picks its color.
const INDICATOR: Record<CategoryCode, string> = {
  food: '*:bg-chart-1',
  transport: '*:bg-chart-2',
  housing: '*:bg-chart-3',
  entertainment: '*:bg-chart-4',
  health: '*:bg-chart-5',
  shopping: '*:bg-chart-6',
  other: '*:bg-chart-7',
}

const DASH = '—'

/**
 * Totals over the whole filter, not the current page (design-spec §4 «Сводка»):
 * Итого, Трат, Средняя трата and the breakdown by category.
 */
export function SummarySection({ filters }: { filters: ExpenseFilters }) {
  const summary = useSummary(filters)
  const isDesktop = useIsDesktop()
  const data = summary.data
  const failed = summary.isError && !data
  const loading = summary.isPending

  if (!isDesktop) return <MobileSummary data={data} loading={loading} failed={failed} />

  return (
    <section aria-label="Сводка" aria-busy={loading} className="flex flex-col gap-3">
      <p className="text-sm text-muted-foreground" aria-live="polite">
        {data
          ? `Сводка по всем ${data.count} ${plural(data.count, 'трате', 'тратам', 'тратам')} по фильтру`
          : failed
            ? 'Не удалось загрузить сводку'
            : 'Считаем сводку…'}
      </p>
      <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-[minmax(380px,5fr)_7fr] lg:gap-6">
        <div className="grid gap-4 md:col-span-3 md:grid-cols-[3fr_2fr_2fr] lg:col-span-1 lg:grid-cols-2">
          <StatCard
            label="Итого"
            className="md:col-span-1 lg:col-span-2"
            valueClassName="text-4xl md:text-[min(40px,11cqi)]"
            value={loading ? undefined : data ? formatRub(data.total) : DASH}
          />
          <StatCard label="Трат" value={loading ? undefined : data ? String(data.count) : DASH} />
          <StatCard label="Средняя трата" value={loading ? undefined : data ? formatRub(average(data)) : DASH} />
        </div>
        <CategoryBreakdown data={data} loading={loading} failed={failed} />
      </div>
    </section>
  )
}

function StatCard({
  label,
  value,
  className,
  valueClassName = 'text-[min(28px,13cqi)]',
}: {
  label: string
  value?: string
  className?: string
  valueClassName?: string
}) {
  return (
    // The value scales with the card width (cqi): three cards in a row fit on a 768px tablet.
    <Card className={cn('@container', className)}>
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        {value === undefined ? (
          <Skeleton className="h-10 w-40" />
        ) : (
          <CardTitle className={cn('font-semibold tracking-tight whitespace-nowrap tabular-nums', valueClassName)}>
            {value}
          </CardTitle>
        )}
      </CardHeader>
    </Card>
  )
}

function CategoryBreakdown({ data, loading, failed }: { data?: Summary; loading: boolean; failed: boolean }) {
  const rows = data?.by_category ?? []
  const max = Math.max(0, ...rows.map((r) => r.total))
  return (
    <Card className="md:col-span-3 lg:col-span-1">
      <CardHeader>
        <CardTitle>По категориям</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <ul className="flex flex-col gap-3" aria-hidden>
            {Array.from({ length: 7 }, (_, i) => (
              <li key={i} className="grid grid-cols-[148px_1fr_112px_40px] items-center gap-3">
                <Skeleton className="h-7 w-28" />
                <Skeleton className="h-2" />
                <Skeleton className="ml-auto h-4 w-20" />
                <Skeleton className="ml-auto h-4 w-8" />
              </li>
            ))}
          </ul>
        ) : failed ? (
          <p className="text-muted-foreground">{DASH}</p>
        ) : rows.length === 0 ? (
          <p className="text-muted-foreground">За выбранный период трат нет</p>
        ) : (
          <ul className="flex flex-col gap-3">
            {rows.map((row) => {
              const cat = CATEGORIES[row.category]
              const Icon = cat.icon
              const pct = share(row.total, data?.total ?? 0)
              return (
                <li key={row.category} className="grid grid-cols-[148px_1fr_112px_40px] items-center gap-3">
                  <span className="flex min-w-0 items-center gap-2">
                    <span className={cn('grid size-7 shrink-0 place-items-center rounded-full', cat.badge)}>
                      <Icon className="size-3.5" aria-hidden />
                    </span>
                    <span className="truncate">{cat.label}</span>
                  </span>
                  <Progress
                    value={max > 0 ? (row.total / max) * 100 : 0}
                    className={cn('h-2', INDICATOR[row.category])}
                    aria-label={`${cat.label}: ${formatRub(row.total)}`}
                  />
                  <span className="text-right font-medium whitespace-nowrap tabular-nums">{formatRub(row.total)}</span>
                  <span className="text-right text-sm text-muted-foreground tabular-nums">{pct}%</span>
                </li>
              )
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}
