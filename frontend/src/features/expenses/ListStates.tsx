import { CloudOff, Plus, Receipt, RefreshCw, RotateCcw } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Empty, EmptyContent, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'

/** «За выбранный период трат нет» with a reset (only for non-default filters) and «Добавить трату». */
export function NoExpenses({ onResetFilters, onAdd }: { onResetFilters?: () => void; onAdd?: () => void }) {
  return (
    <Empty className="rounded-lg border">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Receipt />
        </EmptyMedia>
        <EmptyTitle>За выбранный период трат нет</EmptyTitle>
      </EmptyHeader>
      <EmptyContent className="flex-row flex-wrap justify-center">
        {onResetFilters && (
          <Button variant="outline" className="h-11 md:h-9" onClick={onResetFilters}>
            <RotateCcw />
            Сбросить фильтры
          </Button>
        )}
        <Button className="h-11 md:h-9" onClick={onAdd}>
          <Plus />
          Добавить трату
        </Button>
      </EmptyContent>
    </Empty>
  )
}

/** «Не удалось загрузить траты» with a retry. */
export function LoadFailed({ onRetry }: { onRetry: () => void }) {
  return (
    <Empty className="rounded-lg border" role="alert">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <CloudOff className="text-destructive" />
        </EmptyMedia>
        <EmptyTitle>Не удалось загрузить траты</EmptyTitle>
      </EmptyHeader>
      <EmptyContent>
        <Button variant="outline" className="h-11 md:h-9" onClick={onRetry}>
          <RefreshCw />
          Повторить
        </Button>
      </EmptyContent>
    </Empty>
  )
}
