import { ArrowDown, ArrowUp, ArrowUpDown, Pencil, Trash2 } from 'lucide-react'
import { useEffect, type ReactNode } from 'react'

import type { Expense } from '@/api/client'
import { PAGE_SIZE, useExpenses } from '@/api/hooks'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { cn } from '@/lib/utils'
import { CATEGORIES } from '@/shared/categories'
import { formatDate } from '@/shared/dates'
import { formatRub } from '@/shared/money'
import { initials } from '@/shared/people'

import { LoadFailed, NoExpenses } from './ListStates'
import { TablePagination } from './TablePagination'
import { TruncatedText } from './TruncatedText'
import type { ExpensesView, SortKey, ViewPatch } from './urlState'

type Actions = {
  onEdit?: (e: Expense) => void
  onDelete?: (e: Expense) => void
}

type Props = Actions & {
  view: ExpensesView
  onChange: (patch: ViewPatch) => void
  /** Shown in the empty state when filters differ from the defaults. */
  onResetFilters?: () => void
  onAdd?: () => void
}

type Ctx = Actions & { view: ExpensesView; onChange: (patch: ViewPatch) => void }

type Column = {
  id: 'spent_at' | 'user' | 'category' | 'amount' | 'comment' | 'actions'
  /** Width per design-spec §4: Дата 140 · Кто 176 · Категория 176 · Сумма 160 · Комментарий · 104. */
  width: string
  sortKey?: SortKey
  header: (ctx: Ctx) => ReactNode
  cell: (e: Expense, ctx: Ctx) => ReactNode
}

// Static column list. Sorting and paging happen on the server, so a table
// library would add nothing here.
const COLUMNS: Column[] = [
  {
    id: 'spent_at',
    width: 'w-[140px]',
    sortKey: 'spent_at',
    header: (ctx) => <SortButton ctx={ctx} sortKey="spent_at" label="Дата" />,
    cell: (e) => <span className="tabular-nums">{formatDate(e.spent_at)}</span>,
  },
  {
    id: 'user',
    width: 'w-[176px]',
    header: () => 'Кто',
    cell: (e) => <UserCell name={e.user_name} />,
  },
  {
    id: 'category',
    width: 'w-[176px]',
    header: () => 'Категория',
    cell: (e) => <CategoryBadge expense={e} />,
  },
  {
    id: 'amount',
    width: 'w-[160px] text-right',
    sortKey: 'amount',
    header: (ctx) => <SortButton ctx={ctx} sortKey="amount" label="Сумма" alignEnd />,
    cell: (e) => <div className="text-right font-medium whitespace-nowrap tabular-nums">{formatRub(e.amount)}</div>,
  },
  {
    id: 'comment',
    width: '',
    header: () => 'Комментарий',
    cell: (e) => (e.comment ? <TruncatedText text={e.comment} /> : <span className="text-muted-foreground">—</span>),
  },
  {
    id: 'actions',
    width: 'w-[104px]',
    header: () => <span className="sr-only">Действия</span>,
    cell: (e, ctx) => <RowActions expense={e} ctx={ctx} />,
  },
]

/** Expenses of the current filter, one page at a time, sorted on the server. */
export function ExpensesTable({ view, onChange, onResetFilters, onAdd, onEdit, onDelete }: Props) {
  const list = useExpenses({
    userId: view.userId,
    from: view.from,
    to: view.to,
    category: view.category,
    sort: view.sort,
    order: view.order,
    page: view.page,
  })
  const ctx: Ctx = { view, onChange, onEdit, onDelete }

  // A page past the end (the last expense on it was deleted, or an old link):
  // go to the last page that has expenses.
  const lastPage = list.data ? Math.max(1, Math.ceil(list.data.total / PAGE_SIZE)) : undefined
  useEffect(() => {
    if (lastPage !== undefined && !list.isPlaceholderData && view.page > lastPage) {
      onChange({ page: lastPage })
    }
  }, [lastPage, list.isPlaceholderData, view.page, onChange])

  if (list.isError && !list.data) {
    return <LoadFailed onRetry={() => void list.refetch()} />
  }
  if (list.data && list.data.total === 0) {
    return <NoExpenses onResetFilters={onResetFilters} onAdd={onAdd} />
  }

  const loading = list.isPending
  const ariaSort = (c: Column) =>
    !c.sortKey ? undefined : c.sortKey !== view.sort ? 'none' : view.order === 'asc' ? 'ascending' : 'descending'

  return (
    <div className="flex flex-col gap-3">
      <div className="overflow-hidden rounded-lg border bg-card">
        <Table className="min-w-[900px] table-fixed" aria-busy={loading}>
          <TableHeader className="bg-muted">
            <TableRow className="hover:bg-transparent">
              {COLUMNS.map((c) => (
                <TableHead key={c.id} className={c.width} aria-sort={ariaSort(c)}>
                  {c.header(ctx)}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody className={cn(list.isPlaceholderData && 'opacity-60 transition-opacity')}>
            {loading
              ? Array.from({ length: 6 }, (_, i) => (
                  <TableRow key={i} className="h-[52px]">
                    {COLUMNS.map((c) => (
                      <TableCell key={c.id}>
                        <Skeleton className="h-4 w-full max-w-28" />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              : list.data?.items.map((e) => (
                  <TableRow key={e.id} className="group h-[52px] hover:bg-accent">
                    {COLUMNS.map((c) => (
                      <TableCell key={c.id} className="overflow-hidden">
                        {c.cell(e, ctx)}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
          </TableBody>
        </Table>
      </div>
      {list.data && (
        <TablePagination
          view={view}
          total={list.data.total}
          pageSize={PAGE_SIZE}
          onPage={(page) => onChange({ page })}
        />
      )}
    </div>
  )
}

function SortButton({
  ctx,
  sortKey,
  label,
  alignEnd,
}: {
  ctx: Ctx
  sortKey: SortKey
  label: string
  alignEnd?: boolean
}) {
  const { view, onChange } = ctx
  const active = view.sort === sortKey
  const Icon = !active ? ArrowUpDown : view.order === 'asc' ? ArrowUp : ArrowDown
  return (
    <div className={cn('flex', alignEnd && 'justify-end')}>
      <Button
        variant="ghost"
        size="sm"
        className={alignEnd ? '-mr-2.5' : '-ml-2.5'}
        // A new column starts descending: newest dates, largest amounts.
        onClick={() =>
          onChange(active ? { order: view.order === 'asc' ? 'desc' : 'asc' } : { sort: sortKey, order: 'desc' })
        }
      >
        {label}
        <Icon className={cn(!active && 'text-muted-foreground')} />
      </Button>
    </div>
  )
}

function UserCell({ name }: { name: string }) {
  return (
    <span className="flex min-w-0 items-center gap-2">
      <Avatar className="size-7">
        <AvatarFallback className="bg-secondary text-xs">{initials(name)}</AvatarFallback>
      </Avatar>
      <span className="truncate" title={name}>
        {name}
      </span>
    </span>
  )
}

function CategoryBadge({ expense }: { expense: Expense }) {
  const cat = CATEGORIES[expense.category]
  const Icon = cat.icon
  return (
    <Badge variant="secondary" className={cn('h-[22px] gap-1 rounded-sm border-0', cat.badge)}>
      <Icon className="size-3.5" aria-hidden />
      {cat.label}
    </Badge>
  )
}

function RowActions({ expense, ctx }: { expense: Expense; ctx: Ctx }) {
  const what = `${expense.comment ?? CATEGORIES[expense.category].label}, ${formatRub(expense.amount)}`
  // Muted until the row is hovered or focused, the bin turns red on hover.
  const quiet = 'size-8 text-muted-foreground group-focus-within:text-foreground group-hover:text-foreground'
  return (
    <div className="flex justify-end gap-1">
      <Button
        variant="ghost"
        size="icon"
        className={quiet}
        aria-label={`Изменить: ${what}`}
        onClick={() => ctx.onEdit?.(expense)}
      >
        <Pencil />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        className={cn(quiet, 'hover:text-destructive')}
        aria-label={`Удалить: ${what}`}
        onClick={() => ctx.onDelete?.(expense)}
      >
        <Trash2 />
      </Button>
    </div>
  )
}
