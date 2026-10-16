import { Car, HeartPulse, House, Plus, Shapes, ShoppingBag, Ticket, Utensils } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ThemeToggle } from '@/theme/ThemeToggle'

// Temporary theme check page (FE-01): shows that tokens, radii and the Onest
// font match the design. Replaced by the expenses page in FE-03.
const sample = [
  { label: 'Еда', icon: Utensils, badge: 'bg-chart-1-soft text-chart-1-ink' },
  { label: 'Транспорт', icon: Car, badge: 'bg-chart-2-soft text-chart-2-ink' },
  { label: 'Жильё', icon: House, badge: 'bg-chart-3-soft text-chart-3-ink' },
  { label: 'Развлечения', icon: Ticket, badge: 'bg-chart-4-soft text-chart-4-ink' },
  { label: 'Здоровье', icon: HeartPulse, badge: 'bg-chart-5-soft text-chart-5-ink' },
  { label: 'Покупки', icon: ShoppingBag, badge: 'bg-chart-6-soft text-chart-6-ink' },
  { label: 'Другое', icon: Shapes, badge: 'bg-chart-7-soft text-chart-7-ink' },
]

export function App() {
  return (
    <div className="mx-auto max-w-[1280px] px-4 py-6 md:px-6 lg:px-8">
      <header className="mb-6 flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold md:text-2xl">Семейные расходы</h1>
        <div className="flex items-center gap-2">
          <ThemeToggle />
          <Button>
            <Plus /> Добавить трату
          </Button>
        </div>
      </header>

      <Card>
        <CardHeader>
          <CardDescription>Итого за период</CardDescription>
          <CardTitle className="text-[40px] font-semibold tracking-tight tabular-nums">48&nbsp;720&nbsp;₽</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          {sample.map(({ label, icon: Icon, badge }) => (
            <Badge key={label} variant="secondary" className={`h-[22px] gap-1 rounded-sm border-0 ${badge}`}>
              <Icon className="size-3.5" />
              {label}
            </Badge>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
