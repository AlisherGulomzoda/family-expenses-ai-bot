import type { ExpenseFilters as Filters } from '@/api/hooks'

import { ExpenseFilters } from './ExpenseFilters'
import { ExpensesTable } from './ExpensesTable'
import { PageHeader } from './PageHeader'
import { SummarySection } from './SummarySection'
import { defaultPeriod, hasDefaultFilters, useExpensesView } from './urlState'

/** The only screen of the panel: filters, summary and the expenses list. */
export function ExpensesPage() {
  const [view, update] = useExpensesView()
  const filters: Filters = { userId: view.userId, from: view.from, to: view.to, category: view.category }
  const resetFilters = () => update({ userId: undefined, category: undefined, ...defaultPeriod() })

  return (
    <div className="mx-auto flex max-w-[1280px] flex-col gap-6 px-4 py-6 md:px-6 lg:px-8">
      <PageHeader />
      <ExpenseFilters view={view} onChange={update} />
      <SummarySection filters={filters} />
      {/* Adding, editing (FE-06) and deleting (FE-07) are wired in the next tickets. */}
      <ExpensesTable
        view={view}
        onChange={update}
        onResetFilters={hasDefaultFilters(view) ? undefined : resetFilters}
      />
    </div>
  )
}
