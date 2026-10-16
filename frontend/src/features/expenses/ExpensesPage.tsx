import { useState } from 'react'

import type { Expense } from '@/api/client'
import type { ExpenseFilters as Filters } from '@/api/hooks'
import { useIsDesktop } from '@/shared/useMediaQuery'

import { DeleteExpenseDialog } from './DeleteExpenseDialog'
import { ExpenseCards } from './ExpenseCards'
import { ExpenseFilters } from './ExpenseFilters'
import { ExpenseFormDialog, type Editor } from './ExpenseFormDialog'
import { ExpensesTable } from './ExpensesTable'
import { MobileFilters } from './MobileFilters'
import { FloatingAddButton, PageHeader } from './PageHeader'
import { SummarySection } from './SummarySection'
import { defaultPeriod, hasDefaultFilters, useExpensesView } from './urlState'

/**
 * The only screen of the panel: filters, summary and the expenses list.
 * From 768px: filter bar and table; below: filter drawer, cards and a floating «+».
 */
export function ExpensesPage() {
  const isDesktop = useIsDesktop()
  const [view, update] = useExpensesView()
  const [editor, setEditor] = useState<Editor | null>(null)
  const [toDelete, setToDelete] = useState<Expense | null>(null)
  const filters: Filters = { userId: view.userId, from: view.from, to: view.to, category: view.category }
  const add = () => setEditor({ mode: 'create' })
  const listProps = {
    view,
    onResetFilters: hasDefaultFilters(view)
      ? undefined
      : () => update({ userId: undefined, category: undefined, ...defaultPeriod() }),
    onAdd: add,
    onEdit: (expense: Expense) => setEditor({ mode: 'edit', expense }),
    onDelete: setToDelete,
  }

  return (
    <div className="mx-auto flex max-w-[1280px] flex-col gap-4 px-4 py-4 md:gap-6 md:px-6 md:py-6 lg:px-8">
      {isDesktop ? (
        <>
          <PageHeader onAdd={add} />
          <ExpenseFilters view={view} onChange={update} />
        </>
      ) : (
        // Phone: the header and the filter row stay on top while scrolling.
        <div className="sticky top-0 z-30 -mx-4 -mt-4 flex flex-col gap-3 border-b bg-background/95 px-4 pt-4 pb-3 backdrop-blur">
          <PageHeader compact />
          <MobileFilters view={view} onChange={update} />
        </div>
      )}
      <SummarySection filters={filters} />
      {isDesktop ? <ExpensesTable {...listProps} onChange={update} /> : <ExpenseCards {...listProps} />}
      {!isDesktop && <FloatingAddButton onAdd={add} />}
      <ExpenseFormDialog editor={editor} onClose={() => setEditor(null)} filterUserId={view.userId} />
      <DeleteExpenseDialog expense={toDelete} onClose={() => setToDelete(null)} />
    </div>
  )
}
