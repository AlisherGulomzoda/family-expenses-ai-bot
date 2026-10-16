import { useState } from 'react'

import type { Expense } from '@/api/client'
import type { ExpenseFilters as Filters } from '@/api/hooks'

import { DeleteExpenseDialog } from './DeleteExpenseDialog'
import { ExpenseFilters } from './ExpenseFilters'
import { ExpenseFormDialog, type Editor } from './ExpenseFormDialog'
import { ExpensesTable } from './ExpensesTable'
import { PageHeader } from './PageHeader'
import { SummarySection } from './SummarySection'
import { defaultPeriod, hasDefaultFilters, useExpensesView } from './urlState'

/** The only screen of the panel: filters, summary and the expenses list. */
export function ExpensesPage() {
  const [view, update] = useExpensesView()
  const [editor, setEditor] = useState<Editor | null>(null)
  const [toDelete, setToDelete] = useState<Expense | null>(null)
  const filters: Filters = { userId: view.userId, from: view.from, to: view.to, category: view.category }
  const resetFilters = () => update({ userId: undefined, category: undefined, ...defaultPeriod() })
  const add = () => setEditor({ mode: 'create' })

  return (
    <div className="mx-auto flex max-w-[1280px] flex-col gap-6 px-4 py-6 md:px-6 lg:px-8">
      <PageHeader onAdd={add} />
      <ExpenseFilters view={view} onChange={update} />
      <SummarySection filters={filters} />
      <ExpensesTable
        view={view}
        onChange={update}
        onResetFilters={hasDefaultFilters(view) ? undefined : resetFilters}
        onAdd={add}
        onEdit={(expense) => setEditor({ mode: 'edit', expense })}
        onDelete={setToDelete}
      />
      <ExpenseFormDialog editor={editor} onClose={() => setEditor(null)} filterUserId={view.userId} />
      <DeleteExpenseDialog expense={toDelete} onClose={() => setToDelete(null)} />
    </div>
  )
}
