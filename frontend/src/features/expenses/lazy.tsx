import { lazy, Suspense, useState, type ComponentProps } from 'react'

import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

// Parts that are not on screen when the page opens load on first use: the form
// (zod, react-hook-form), the date picker (react-day-picker) and the phone filter
// drawer (vaul) stay out of the initial bundle.

const LazyForm = lazy(() => import('./ExpenseFormDialog').then((m) => ({ default: m.ExpenseFormDialog })))
const LazyCalendar = lazy(() => import('./PeriodCalendar').then((m) => ({ default: m.PeriodCalendar })))
const LazyDrawer = lazy(() => import('./FiltersDrawer').then((m) => ({ default: m.FiltersDrawer })))

/** Create/edit dialog, loaded when first opened and kept mounted so it can animate closing. */
export function ExpenseFormDialog(props: ComponentProps<typeof LazyForm>) {
  const used = useOpenedOnce(props.editor !== null)
  return used ? (
    <Suspense>
      <LazyForm {...props} />
    </Suspense>
  ) : null
}

/** Phone filter drawer, loaded when first opened. */
export function FiltersDrawer(props: ComponentProps<typeof LazyDrawer>) {
  const used = useOpenedOnce(props.open)
  return used ? (
    <Suspense>
      <LazyDrawer {...props} />
    </Suspense>
  ) : null
}

/** Period calendar with a placeholder of its size while the picker loads. */
export function PeriodCalendar(props: ComponentProps<typeof LazyCalendar>) {
  return (
    <Suspense
      fallback={<Skeleton aria-hidden className={cn('m-3 h-72', props.numberOfMonths === 2 ? 'w-[30rem]' : 'w-60')} />}
    >
      <LazyCalendar {...props} />
    </Suspense>
  )
}

/** True from the first time `open` is true. */
function useOpenedOnce(open: boolean) {
  const [opened, setOpened] = useState(open)
  // Adjusting state while rendering: https://react.dev/learn/you-might-not-need-an-effect
  if (open && !opened) {
    setOpened(true)
  }
  return opened || open
}
