import { zodResolver } from '@hookform/resolvers/zod'
import { addDays } from 'date-fns'
import { AlertCircle, CalendarDays, Check } from 'lucide-react'
import { useEffect, useState } from 'react'
import { ru } from 'react-day-picker/locale'
import { Controller, useForm, type Resolver, type UseFormReturn } from 'react-hook-form'
import { toast } from 'sonner'

import { ApiError, type Expense } from '@/api/client'
import { useCreateExpense, useUpdateExpense, useUsers } from '@/api/hooks'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from '@/components/ui/field'
import { InputGroup, InputGroupAddon, InputGroupInput } from '@/components/ui/input-group'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { cn } from '@/lib/utils'
import { CATEGORIES, CATEGORY_CODES, isCategoryCode, type CategoryCode } from '@/shared/categories'
import { formatDayLabel, fromISODate, toISODate } from '@/shared/dates'
import { formatRub } from '@/shared/money'
import { initials } from '@/shared/people'

import {
  API_FIELDS,
  defaultMember,
  editValues,
  formSchema,
  newValues,
  rememberMember,
  toBody,
  type FormValues,
} from './expenseForm'

/** What the dialog edits: a new expense or an existing one. */
export type Editor = { mode: 'create' } | { mode: 'edit'; expense: Expense }

type Props = {
  editor: Editor | null
  onClose: () => void
  /** Member of the current filter: the default for a new expense. */
  filterUserId?: number
  /** «Удалить» in the edit dialog (the confirmation is FE-07). */
  onDelete?: (e: Expense) => void
}

/** Create and edit dialog (design: ExpenseDialog, DialogNew, DialogErrors, DialogEdit). */
export function ExpenseFormDialog({ editor, onClose, filterUserId, onDelete }: Props) {
  return (
    <Dialog open={editor !== null} onOpenChange={(open) => !open && onClose()}>
      {editor && (
        <DialogContent className="rounded-xl sm:max-w-[520px]">
          {/* A fresh form per opened expense: no state leaks between dialogs. */}
          <ExpenseForm
            key={editor.mode === 'edit' ? `edit-${editor.expense.id}` : 'create'}
            editor={editor}
            filterUserId={filterUserId}
            onDone={onClose}
            onDelete={onDelete}
          />
        </DialogContent>
      )}
    </Dialog>
  )
}

type FormProps = {
  editor: Editor
  filterUserId?: number
  onDone: () => void
  onDelete?: (e: Expense) => void
}

function ExpenseForm({ editor, filterUserId, onDone, onDelete }: FormProps) {
  const users = useUsers()
  const create = useCreateExpense()
  const update = useUpdateExpense()
  const editing = editor.mode === 'edit' ? editor.expense : undefined

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema) as Resolver<FormValues>,
    defaultValues: editing ? editValues(editing) : newValues(defaultMember(users.data, filterUserId)),
  })

  // Family members may arrive after the dialog opened.
  useEffect(() => {
    if (!editing && form.getValues('userId') === '') {
      const id = defaultMember(users.data, filterUserId)
      if (id !== undefined) form.setValue('userId', String(id))
    }
  }, [users.data, editing, filterUserId, form])

  const saving = form.formState.isSubmitting
  // A failed save is shown inside the dialog, not as a toast: while the modal
  // is open, everything outside it (toasts too) ignores the pointer, and a
  // click outside would close the dialog and lose what was typed.
  const [saveError, setSaveError] = useState<string | null>(null)

  const save = async (values: FormValues) => {
    const body = toBody(values)
    setSaveError(null)
    try {
      const saved = editing ? await update.mutateAsync({ id: editing.id, body }) : await create.mutateAsync(body)
      rememberMember(saved.user_id)
      const what = saved.comment ?? CATEGORIES[saved.category].label
      toast.success(editing ? 'Изменения сохранены' : 'Трата добавлена', {
        description: `${capitalize(what)} · ${formatRub(saved.amount)} · ${saved.user_name}`,
        duration: 4000,
      })
      onDone()
    } catch (err) {
      if (!showFieldErrors(err, form)) {
        setSaveError(
          err instanceof ApiError && err.status !== 0 ? err.message : 'Нет связи с сервером. Попробуйте ещё раз.',
        )
      }
    }
  }

  return (
    <form onSubmit={form.handleSubmit(save)} noValidate className="flex flex-col gap-5">
      <DialogHeader>
        <DialogTitle className="text-xl">{editing ? 'Редактирование траты' : 'Новая трата'}</DialogTitle>
        <DialogDescription>
          {!editing
            ? 'Траты из Telegram-бота появляются здесь сами. Добавьте то, что не записали в бота.'
            : editing.source === 'bot'
              ? 'Записано через Telegram-бота'
              : 'Добавлено в веб-панели'}
        </DialogDescription>
      </DialogHeader>

      <FieldGroup className="gap-4">
        <Controller
          control={form.control}
          name="userId"
          render={({ field, fieldState }) => (
            <Field data-invalid={fieldState.invalid}>
              <FieldLabel htmlFor="expense-user">Член семьи</FieldLabel>
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger id="expense-user" className="w-full" aria-invalid={fieldState.invalid}>
                  <SelectValue placeholder="Выберите" />
                </SelectTrigger>
                <SelectContent>
                  {users.data?.map((u) => (
                    <SelectItem key={u.id} value={String(u.id)}>
                      <Avatar className="size-5">
                        <AvatarFallback className="bg-secondary text-[10px]">{initials(u.name)}</AvatarFallback>
                      </Avatar>
                      <span className="truncate">{u.name}</span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <ErrorText error={fieldState.error?.message} />
            </Field>
          )}
        />

        <Controller
          control={form.control}
          name="amount"
          render={({ field, fieldState }) => (
            <Field data-invalid={fieldState.invalid}>
              <FieldLabel htmlFor="expense-amount">Сумма, ₽</FieldLabel>
              <InputGroup className="h-14">
                <InputGroupInput
                  id="expense-amount"
                  inputMode="decimal"
                  autoComplete="off"
                  placeholder="0"
                  className="h-14 text-[28px] font-semibold tabular-nums"
                  aria-invalid={fieldState.invalid}
                  aria-describedby="expense-amount-hint"
                  {...field}
                />
                <InputGroupAddon align="inline-end" className="text-xl">
                  ₽
                </InputGroupAddon>
              </InputGroup>
              {fieldState.error ? (
                <ErrorText error={fieldState.error.message} />
              ) : (
                <FieldDescription id="expense-amount-hint">Можно с копейками: 349,90</FieldDescription>
              )}
            </Field>
          )}
        />

        <Controller
          control={form.control}
          name="category"
          render={({ field, fieldState }) => (
            <Field data-invalid={fieldState.invalid}>
              <FieldLabel id="expense-category-label">Категория</FieldLabel>
              <CategoryTiles
                value={field.value}
                onChange={field.onChange}
                invalid={fieldState.invalid}
                labelledBy="expense-category-label"
              />
              <ErrorText error={fieldState.error?.message} />
            </Field>
          )}
        />

        <Controller
          control={form.control}
          name="spentAt"
          render={({ field, fieldState }) => (
            <Field data-invalid={fieldState.invalid}>
              <FieldLabel htmlFor="expense-date">Дата</FieldLabel>
              <DateField id="expense-date" value={field.value} onChange={field.onChange} invalid={fieldState.invalid} />
              <ErrorText error={fieldState.error?.message} />
            </Field>
          )}
        />

        <Controller
          control={form.control}
          name="comment"
          render={({ field, fieldState }) => (
            <Field data-invalid={fieldState.invalid}>
              <FieldLabel htmlFor="expense-comment">
                Комментарий <span className="font-normal text-muted-foreground">— необязательно</span>
              </FieldLabel>
              <Textarea
                id="expense-comment"
                rows={2}
                maxLength={500}
                placeholder="Например: кофе с коллегами"
                className="text-base"
                aria-invalid={fieldState.invalid}
                {...field}
              />
              <ErrorText error={fieldState.error?.message} />
            </Field>
          )}
        />
      </FieldGroup>

      {saveError && (
        <div
          role="alert"
          className="flex items-center gap-3 rounded-lg border border-destructive/40 bg-destructive/5 p-3 text-sm"
        >
          <AlertCircle className="size-4 shrink-0 text-destructive" aria-hidden />
          <div className="flex-1">
            <p className="font-medium">Не удалось сохранить</p>
            <p className="text-muted-foreground">{saveError}</p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={saving}
            onClick={() => void form.handleSubmit(save)()}
          >
            Повторить
          </Button>
        </div>
      )}

      <DialogFooter className="items-center gap-2 sm:justify-between">
        {editing && onDelete ? (
          <Button
            type="button"
            variant="ghost"
            className="text-destructive hover:text-destructive sm:mr-auto"
            onClick={() => onDelete(editing)}
          >
            Удалить
          </Button>
        ) : (
          <span />
        )}
        <div className="flex gap-2">
          <Button type="button" variant="secondary" onClick={onDone}>
            Отмена
          </Button>
          <Button type="submit" className="min-w-[136px] disabled:opacity-100" disabled={saving}>
            {saving ? (
              <>
                <Spinner />
                Сохраняем…
              </>
            ) : (
              'Сохранить'
            )}
          </Button>
        </div>
      </DialogFooter>
    </form>
  )
}

function ErrorText({ error }: { error?: string }) {
  if (!error) return null
  return (
    <FieldError className="flex items-center gap-1.5">
      <AlertCircle className="size-4 shrink-0" aria-hidden />
      {error}
    </FieldError>
  )
}

// Selected tile: soft background and a 2px ring in the category's ink color.
const SELECTED: Record<CategoryCode, string> = {
  food: 'data-[state=on]:bg-chart-1-soft data-[state=on]:ring-chart-1-ink',
  transport: 'data-[state=on]:bg-chart-2-soft data-[state=on]:ring-chart-2-ink',
  housing: 'data-[state=on]:bg-chart-3-soft data-[state=on]:ring-chart-3-ink',
  entertainment: 'data-[state=on]:bg-chart-4-soft data-[state=on]:ring-chart-4-ink',
  health: 'data-[state=on]:bg-chart-5-soft data-[state=on]:ring-chart-5-ink',
  shopping: 'data-[state=on]:bg-chart-6-soft data-[state=on]:ring-chart-6-ink',
  other: 'data-[state=on]:bg-chart-7-soft data-[state=on]:ring-chart-7-ink',
}

function CategoryTiles({
  value,
  onChange,
  invalid,
  labelledBy,
}: {
  value: string
  onChange: (v: CategoryCode | '') => void
  invalid: boolean
  labelledBy: string
}) {
  return (
    <ToggleGroup
      type="single"
      variant="outline"
      spacing={2}
      value={value}
      // Clicking the selected tile again keeps it selected.
      onValueChange={(v) => v && isCategoryCode(v) && onChange(v)}
      className="grid w-full grid-cols-4 gap-2"
      aria-labelledby={labelledBy}
      aria-invalid={invalid}
    >
      {CATEGORY_CODES.map((code) => {
        const cat = CATEGORIES[code]
        const Icon = cat.icon
        return (
          <ToggleGroupItem
            key={code}
            value={code}
            className={cn(
              'group/tile relative h-[76px] w-full flex-col gap-1.5 rounded-lg px-1 text-xs data-[state=on]:ring-2',
              SELECTED[code],
              invalid && 'border-destructive',
            )}
          >
            <span className={cn('grid size-8 place-items-center rounded-full', cat.badge)}>
              <Icon className="size-4" aria-hidden />
            </span>
            <span className="truncate">{cat.label}</span>
            <Check
              className="absolute top-1.5 right-1.5 hidden size-3.5 group-data-[state=on]/tile:block"
              aria-hidden
            />
          </ToggleGroupItem>
        )
      })}
    </ToggleGroup>
  )
}

function DateField({
  id,
  value,
  onChange,
  invalid,
}: {
  id: string
  value: string
  onChange: (v: string) => void
  invalid: boolean
}) {
  const [open, setOpen] = useState(false)
  // Latest selectable day, fixed when the field mounts (as on the backend: tomorrow).
  const [tomorrow] = useState(() => addDays(new Date(), 1))
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button id={id} type="button" variant="outline" className="w-full justify-between" aria-invalid={invalid}>
          {formatDayLabel(value)}
          <CalendarDays className="text-muted-foreground" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="single"
          locale={ru}
          selected={fromISODate(value)}
          defaultMonth={fromISODate(value)}
          disabled={{ after: tomorrow }}
          onSelect={(d) => {
            if (d) {
              onChange(toISODate(d))
              setOpen(false)
            }
          }}
        />
      </PopoverContent>
    </Popover>
  )
}

/** Puts 422 field messages under the fields; false if the error is not about fields. */
function showFieldErrors(err: unknown, form: UseFormReturn<FormValues>): boolean {
  if (!(err instanceof ApiError) || err.status !== 422) return false
  let shown = false
  for (const [apiField, message] of Object.entries(err.fields)) {
    const name = API_FIELDS[apiField]
    if (name) {
      form.setError(name, { type: 'server', message }, { shouldFocus: !shown })
      shown = true
    }
  }
  return shown
}

function capitalize(s: string) {
  return s.charAt(0).toUpperCase() + s.slice(1)
}
