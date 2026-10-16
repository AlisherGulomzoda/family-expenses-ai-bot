import type { MouseEvent } from 'react'

import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import { cn } from '@/lib/utils'

import { pageItems } from './paging'
import { applyPatch, serializeView, type ExpensesView } from './urlState'

type Props = { view: ExpensesView; total: number; pageSize: number; onPage: (page: number) => void }

/**
 * «Показано 1–10 из 37» and page links. Links are real URLs (open in a new
 * tab works); a plain click changes the page without reloading.
 */
export function TablePagination({ view, total, pageSize, onPage }: Props) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  const page = Math.min(view.page, pages)
  const first = total === 0 ? 0 : (page - 1) * pageSize + 1
  const last = Math.min(page * pageSize, total)

  const link = (n: number) => ({
    href: serializeView(applyPatch(view, { page: n })) || '?',
    onClick: (e: MouseEvent) => {
      if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return
      e.preventDefault()
      onPage(n)
    },
  })

  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-muted-foreground tabular-nums">
        Показано {first}–{last} из {total}
      </p>
      {pages > 1 && (
        <Pagination className="mx-0 w-auto justify-end">
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious
                text="Назад"
                aria-label="Предыдущая страница"
                aria-disabled={page === 1}
                className={cn(page === 1 && 'pointer-events-none opacity-50')}
                {...link(Math.max(1, page - 1))}
              />
            </PaginationItem>
            {pageItems(page, pages).map((item, i) =>
              item === 'gap' ? (
                <PaginationItem key={`gap-${i}`}>
                  <PaginationEllipsis />
                </PaginationItem>
              ) : (
                <PaginationItem key={item}>
                  <PaginationLink
                    isActive={item === page}
                    aria-label={`Страница ${item}`}
                    className={cn(
                      'tabular-nums',
                      item === page &&
                        'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground',
                    )}
                    {...link(item)}
                  >
                    {item}
                  </PaginationLink>
                </PaginationItem>
              ),
            )}
            <PaginationItem>
              <PaginationNext
                text="Вперёд"
                aria-label="Следующая страница"
                aria-disabled={page === pages}
                className={cn(page === pages && 'pointer-events-none opacity-50')}
                {...link(Math.min(pages, page + 1))}
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      )}
    </div>
  )
}
