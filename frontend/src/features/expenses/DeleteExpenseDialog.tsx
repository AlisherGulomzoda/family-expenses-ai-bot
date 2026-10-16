import { AlertCircle } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { ApiError, type Expense } from '@/api/client'
import { useDeleteExpense } from '@/api/hooks'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Spinner } from '@/components/ui/spinner'
import { cn } from '@/lib/utils'
import { CATEGORIES } from '@/shared/categories'
import { formatDate } from '@/shared/dates'
import { formatRub } from '@/shared/money'
import { capitalize } from '@/shared/text'

type Props = {
  /** The expense to delete; null keeps the dialog closed. */
  expense: Expense | null
  onClose: () => void
  /** Called after a successful delete (e.g. to close the edit dialog too). */
  onDeleted?: () => void
}

/** «Удалить трату?» with the expense itself, so it is clear what goes away (design: DeleteConfirm). */
export function DeleteExpenseDialog({ expense, onClose, onDeleted }: Props) {
  const remove = useDeleteExpense()
  const [error, setError] = useState<string | null>(null)
  const deleting = remove.isPending

  const close = () => {
    if (deleting) return
    setError(null)
    onClose()
  }

  const confirm = async () => {
    if (!expense) return
    setError(null)
    try {
      await remove.mutateAsync(expense)
      const cat = CATEGORIES[expense.category]
      toast.success('Трата удалена', {
        description: `${capitalize(expense.comment ?? cat.label)} · ${formatRub(expense.amount)} · ${expense.user_name}`,
        duration: 4000,
      })
      onDeleted?.()
      onClose()
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        // Already gone (deleted in the bot or another tab): the result is the same.
        toast.success('Трата уже удалена')
        onDeleted?.()
        onClose()
        return
      }
      setError(err instanceof ApiError && err.status !== 0 ? err.message : 'Нет связи с сервером. Попробуйте ещё раз.')
    }
  }

  const cat = expense ? CATEGORIES[expense.category] : undefined
  const Icon = cat?.icon

  return (
    <AlertDialog open={expense !== null} onOpenChange={(open) => !open && close()}>
      <AlertDialogContent className="max-w-[calc(100%-2rem)] sm:max-w-[440px]">
        <AlertDialogHeader>
          <AlertDialogTitle className="text-lg">Удалить трату?</AlertDialogTitle>
          <AlertDialogDescription>Это действие нельзя отменить.</AlertDialogDescription>
        </AlertDialogHeader>

        {expense && cat && Icon && (
          <div className="flex items-center gap-3 rounded-lg bg-muted p-3" data-testid="expense-to-delete">
            <span className={cn('grid size-10 shrink-0 place-items-center rounded-full', cat.badge)}>
              <Icon className="size-5" aria-hidden />
            </span>
            <div className="min-w-0 flex-1">
              <p className="truncate font-medium">{capitalize(expense.comment ?? cat.label)}</p>
              <p className="truncate text-sm text-muted-foreground">
                {cat.label} · {expense.user_name} · {formatDate(expense.spent_at)}
              </p>
            </div>
            <span className="shrink-0 font-semibold whitespace-nowrap tabular-nums">{formatRub(expense.amount)}</span>
          </div>
        )}

        {error && (
          <p role="alert" className="flex items-center gap-2 text-sm text-destructive">
            <AlertCircle className="size-4 shrink-0" aria-hidden />
            Не удалось удалить: {error}
          </p>
        )}

        <AlertDialogFooter>
          {/* Radix focuses «Отмена» first: Enter right after opening does not delete. */}
          <AlertDialogCancel variant="secondary" disabled={deleting}>
            Отмена
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={deleting}
            onClick={(e) => {
              // Keep the dialog open until the server answers.
              e.preventDefault()
              void confirm()
            }}
          >
            {deleting && <Spinner />}
            {deleting ? 'Удаляем…' : error ? 'Повторить' : 'Удалить'}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
