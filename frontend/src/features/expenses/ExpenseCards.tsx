import { MoreHorizontal, Pencil, Trash2 } from 'lucide-react'

import type { Expense } from '@/api/client'
import { PAGE_SIZE, useExpensesInfinite } from '@/api/hooks'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Item, ItemActions, ItemContent, ItemDescription, ItemMedia, ItemTitle } from '@/components/ui/item'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { cn } from '@/lib/utils'
import { CATEGORIES } from '@/shared/categories'
import { formatShortDate } from '@/shared/dates'
import { formatRub } from '@/shared/money'
import { capitalize } from '@/shared/text'

import { LoadFailed, NoExpenses } from './ListStates'
import type { ExpensesView } from './urlState'

type Props = {
  view: ExpensesView
  onResetFilters?: () => void
  onAdd?: () => void
  onEdit?: (e: Expense) => void
  onDelete?: (e: Expense) => void
}

/** Phone list (< 768px): cards, a «⋯» menu per card and «Показать ещё» instead of pages. */
export function ExpenseCards({ view, onResetFilters, onAdd, onEdit, onDelete }: Props) {
  const list = useExpensesInfinite({
    userId: view.userId,
    from: view.from,
    to: view.to,
    category: view.category,
    sort: view.sort,
    order: view.order,
  })

  if (list.isError && !list.data) return <LoadFailed onRetry={() => void list.refetch()} />

  const items = list.data?.pages.flatMap((p) => p.items) ?? []
  const total = list.data?.pages[0]?.total ?? 0
  if (list.data && total === 0) return <NoExpenses onResetFilters={onResetFilters} onAdd={onAdd} />

  const more = Math.min(PAGE_SIZE, total - items.length)

  return (
    // Bottom padding keeps the floating «+» from covering the last card.
    <section aria-label="Траты" aria-busy={list.isPending} className="flex flex-col gap-2 pb-28">
      {list.isPending
        ? Array.from({ length: 6 }, (_, i) => <Skeleton key={i} className="h-[72px] rounded-lg" />)
        : items.map((e) => <ExpenseCard key={e.id} expense={e} onEdit={onEdit} onDelete={onDelete} />)}
      {more > 0 && (
        <Button
          variant="outline"
          size="lg"
          className="mt-2 h-11 w-full"
          disabled={list.isFetchingNextPage}
          onClick={() => void list.fetchNextPage()}
        >
          {list.isFetchingNextPage && <Spinner />}
          Показать ещё {more}
        </Button>
      )}
    </section>
  )
}

function ExpenseCard({
  expense,
  onEdit,
  onDelete,
}: {
  expense: Expense
  onEdit?: (e: Expense) => void
  onDelete?: (e: Expense) => void
}) {
  const cat = CATEGORIES[expense.category]
  const Icon = cat.icon
  const title = capitalize(expense.comment ?? cat.label)
  return (
    <Item variant="outline" className="flex-nowrap bg-card">
      <ItemMedia>
        <span className={cn('grid size-10 place-items-center rounded-full', cat.badge)}>
          <Icon className="size-5" aria-hidden />
        </span>
      </ItemMedia>
      <ItemContent className="min-w-0">
        <ItemTitle className="w-full truncate">{title}</ItemTitle>
        <ItemDescription className="truncate">
          {expense.user_name} · {formatShortDate(expense.spent_at)}
        </ItemDescription>
      </ItemContent>
      <ItemActions className="shrink-0 gap-1">
        <span className="font-medium whitespace-nowrap tabular-nums">{formatRub(expense.amount)}</span>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-11"
              aria-label={`Действия: ${title}, ${formatRub(expense.amount)}`}
            >
              <MoreHorizontal />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem className="h-11" onSelect={() => onEdit?.(expense)}>
              <Pencil />
              Изменить
            </DropdownMenuItem>
            <DropdownMenuItem className="h-11" variant="destructive" onSelect={() => onDelete?.(expense)}>
              <Trash2 />
              Удалить
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </ItemActions>
    </Item>
  )
}
