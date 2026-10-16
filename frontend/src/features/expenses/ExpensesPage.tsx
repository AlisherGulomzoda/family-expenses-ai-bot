import { useSummary } from '@/api/hooks'
import { formatRub } from '@/shared/money'

import { ExpenseFilters } from './ExpenseFilters'
import { PageHeader } from './PageHeader'
import { useExpensesView } from './urlState'

/** The only screen of the panel: filters, summary and the expenses list. */
export function ExpensesPage() {
  const [view, update] = useExpensesView()
  const filters = { userId: view.userId, from: view.from, to: view.to, category: view.category }
  const summary = useSummary(filters)

  return (
    <div className="mx-auto flex max-w-[1280px] flex-col gap-6 px-4 py-6 md:px-6 lg:px-8">
      <PageHeader />
      <ExpenseFilters view={view} onChange={update} />
      {/* Summary cards (FE-04) and the table (FE-05) go here. */}
      <p className="text-muted-foreground" aria-live="polite">
        {summary.data
          ? `Сводка по всем ${summary.data.count} тратам по фильтру: ${formatRub(summary.data.total)}`
          : 'Загружаем траты…'}
      </p>
    </div>
  )
}
