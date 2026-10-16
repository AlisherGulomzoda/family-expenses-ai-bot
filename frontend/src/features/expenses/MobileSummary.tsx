import { ChevronDown } from 'lucide-react'
import { useState } from 'react'

import type { Summary } from '@/api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { CATEGORIES } from '@/shared/categories'
import { formatRub } from '@/shared/money'
import { plural } from '@/shared/plural'

import { average, share } from './stats'

/**
 * Phone summary (< 768px, design: MobileMain): one card with the total,
 * «N трат · в среднем …», a stacked bar of category shares with the three
 * largest named below it, and the full legend behind «По категориям: N».
 */
export function MobileSummary({ data, loading, failed }: { data?: Summary; loading: boolean; failed: boolean }) {
  const [legendOpen, setLegendOpen] = useState(false)
  const rows = data?.by_category ?? []
  const total = data?.total ?? 0

  return (
    <section aria-label="Сводка" aria-busy={loading}>
      <Card>
        <CardContent className="flex flex-col gap-3">
          <div>
            <p className="text-sm text-muted-foreground">Итого</p>
            {loading ? (
              <Skeleton className="mt-1 h-10 w-44" />
            ) : (
              <p className="text-4xl font-semibold tracking-tight whitespace-nowrap tabular-nums">
                {data ? formatRub(data.total) : '—'}
              </p>
            )}
            <p className="text-sm text-muted-foreground" aria-live="polite">
              {data
                ? `${data.count} ${plural(data.count, 'трата', 'траты', 'трат')} · в среднем ${formatRub(average(data))}`
                : failed
                  ? 'Не удалось загрузить сводку'
                  : 'Считаем сводку…'}
            </p>
          </div>

          {rows.length > 0 && (
            <>
              {/* Stacked bar of shares; a 2px card-colored gap separates the segments. */}
              <div className="flex h-3 w-full gap-0.5 overflow-hidden rounded-full" aria-hidden>
                {rows.map((r) => (
                  <span
                    key={r.category}
                    className={cn('h-full first:rounded-l-full last:rounded-r-full', CATEGORIES[r.category].fill)}
                    style={{ width: `${(r.total / total) * 100}%` }}
                  />
                ))}
              </div>
              <p className="text-sm">
                {rows
                  .slice(0, 3)
                  .map((r) => `${CATEGORIES[r.category].label} ${share(r.total, total)}%`)
                  .join(' · ')}
              </p>

              <Collapsible open={legendOpen} onOpenChange={setLegendOpen}>
                <CollapsibleTrigger asChild>
                  <Button variant="ghost" className="h-11 w-full justify-between px-2">
                    {legendOpen ? 'Скрыть категории' : `По категориям: ${rows.length}`}
                    <ChevronDown className={cn('transition-transform', legendOpen && 'rotate-180')} />
                  </Button>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <ul className="flex flex-col gap-2 pt-2">
                    {rows.map((r) => {
                      const cat = CATEGORIES[r.category]
                      return (
                        <li key={r.category} className="flex items-center gap-2 text-sm">
                          <span className={cn('size-2.5 shrink-0 rounded-full', cat.fill)} aria-hidden />
                          <span className="flex-1 truncate">{cat.label}</span>
                          <span className="font-medium whitespace-nowrap tabular-nums">{formatRub(r.total)}</span>
                          <span className="w-10 text-right text-muted-foreground tabular-nums">
                            {share(r.total, total)}%
                          </span>
                        </li>
                      )
                    })}
                  </ul>
                </CollapsibleContent>
              </Collapsible>
            </>
          )}
        </CardContent>
      </Card>
    </section>
  )
}
