> Локальная копия спецификации из Claude Design (снята 2026-10-16) — чтобы контекст не зависел от доступа к ссылке.
> Источник: https://claude.ai/artifact/LP5FghRogpZTEqpKgW98Qr · Макеты (18 артбордов): https://claude.ai/artifact/W7ZRVshSuWBLaJFiq3VCMj · Дизайн-система (токены, шрифты Onest): https://claude.ai/code/artifact/48d056c2-5924-4259-92c5-5f30842fc16e
> Расхождения с принятыми решениями и их разрешение — в `backlog/B3-frontend.md` §9.
# Семейные расходы — спецификация для разработки

2026-10-16 · @Alisher

Всё собирается из стандартных компонентов shadcn/ui на Tailwind v4: своя только тема и 21 токен категорий. Макеты — на холсте [Семейные расходы — главный экран](https://claude.ai/artifact/W7ZRVshSuWBLaJFiq3VCMj), токены и шрифты — в [дизайн-системе](https://claude.ai/code/artifact/48d056c2-5924-4259-92c5-5f30842fc16e).

## 1. Тема: globals.css

Скопируйте блок целиком вместо темы, которую сгенерировал `npx shadcn init`. Тёмная тема включается классом `.dark` на `<html>` (next-themes, `attribute="class"`). Контраст всех пар текст/фон ≥ 4,5:1 в обеих темах; рамки полей и кольцо фокуса ≥ 3:1.

```css
@import "tailwindcss";
@import "tw-animate-css";

@custom-variant dark (&:is(.dark *));

/* Onest 400/500/600/700 — файлы fonts/Onest-*.woff2 из дизайн-системы, положить в /public/fonts */
@font-face { font-family: "Onest"; src: url("/fonts/Onest-400.woff2") format("woff2"); font-weight: 400; font-display: swap; }
@font-face { font-family: "Onest"; src: url("/fonts/Onest-500.woff2") format("woff2"); font-weight: 500; font-display: swap; }
@font-face { font-family: "Onest"; src: url("/fonts/Onest-600.woff2") format("woff2"); font-weight: 600; font-display: swap; }
@font-face { font-family: "Onest"; src: url("/fonts/Onest-700.woff2") format("woff2"); font-weight: 700; font-display: swap; }

:root {
  --radius: 0.625rem;
  --background: #f6f7f5;
  --foreground: #172120;
  --card: #ffffff;
  --card-foreground: #172120;
  --popover: #ffffff;
  --popover-foreground: #172120;
  --primary: #1d6b5a;
  --primary-foreground: #ffffff;
  --secondary: #eceeeb;
  --secondary-foreground: #172120;
  --muted: #eef0ed;
  --muted-foreground: #5a6662;
  --accent: #e3eeea;
  --accent-foreground: #16463b;
  --destructive: #c0392b;
  --destructive-foreground: #ffffff;
  --border: #dfe3e0;
  --input: #85908c;
  --ring: #1d6b5a;
  /* категории: заливка / фон бейджа / текст бейджа */
  --chart-1: #d9a11c; --chart-1-soft: #faf2df; --chart-1-ink: #846211;
  --chart-2: #3f5fbd; --chart-2-soft: #e4e9f6; --chart-2-ink: #3d5cb7;
  --chart-3: #2f9a80; --chart-3-soft: #e2f1ed; --chart-3-ink: #22705d;
  --chart-4: #8b4d86; --chart-4-soft: #efe6ee; --chart-4-ink: #884b83;
  --chart-5: #b8461b; --chart-5-soft: #f5e5df; --chart-5-ink: #a74019;
  --chart-6: #4fb4e8; --chart-6-soft: #e6f4fc; --chart-6-ink: #2f6c8b;
  --chart-7: #4a4044; --chart-7-soft: #e6e4e5; --chart-7-ink: #4a4044;
}

.dark {
  --background: #0f1312;
  --foreground: #e7ecea;
  --card: #171d1c;
  --card-foreground: #e7ecea;
  --popover: #1c2322;
  --popover-foreground: #e7ecea;
  --primary: #5cc2a6;
  --primary-foreground: #06231c;
  --secondary: #232b29;
  --secondary-foreground: #e7ecea;
  --muted: #1f2725;
  --muted-foreground: #9ca8a4;
  --accent: #1f3530;
  --accent-foreground: #c9ebe1;
  --destructive: #f07a6a;
  --destructive-foreground: #2b0905;
  --border: #2a3331;
  --input: #66736f;
  --ring: #5cc2a6;
  --chart-1: #f5cf4f; --chart-1-soft: #434126; --chart-1-ink: #f5cf4f;
  --chart-2: #4a85dd; --chart-2-soft: #213243; --chart-2-ink: #75a2e5;
  --chart-3: #2bbb93; --chart-3-soft: #1b3d34; --chart-3-ink: #31bd96;
  --chart-4: #c49af5; --chart-4-soft: #3a3647; --chart-4-ink: #c49af5;
  --chart-5: #ee8a2a; --chart-5-soft: #42331f; --chart-5-ink: #ef8f33;
  --chart-6: #ace3f1; --chart-6-soft: #354547; --chart-6-ink: #ace3f1;
  --chart-7: #7a6c6e; --chart-7-soft: #2b2d2c; --chart-7-ink: #a39a9b;
}

@theme inline {
  --font-sans: "Onest", "Segoe UI", Roboto, system-ui, sans-serif;
  --radius-sm: calc(var(--radius) - 4px);   /* 6px: бейджи */
  --radius-md: calc(var(--radius) - 2px);   /* 8px: кнопки, поля */
  --radius-lg: var(--radius);               /* 10px: карточки, поповеры */
  --radius-xl: calc(var(--radius) + 4px);   /* 14px: диалоги, шторка */
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-card: var(--card);
  --color-card-foreground: var(--card-foreground);
  --color-popover: var(--popover);
  --color-popover-foreground: var(--popover-foreground);
  --color-primary: var(--primary);
  --color-primary-foreground: var(--primary-foreground);
  --color-secondary: var(--secondary);
  --color-secondary-foreground: var(--secondary-foreground);
  --color-muted: var(--muted);
  --color-muted-foreground: var(--muted-foreground);
  --color-accent: var(--accent);
  --color-accent-foreground: var(--accent-foreground);
  --color-destructive: var(--destructive);
  --color-destructive-foreground: var(--destructive-foreground);
  --color-border: var(--border);
  --color-input: var(--input);
  --color-ring: var(--ring);
  --color-chart-1: var(--chart-1); --color-chart-1-soft: var(--chart-1-soft); --color-chart-1-ink: var(--chart-1-ink);
  --color-chart-2: var(--chart-2); --color-chart-2-soft: var(--chart-2-soft); --color-chart-2-ink: var(--chart-2-ink);
  --color-chart-3: var(--chart-3); --color-chart-3-soft: var(--chart-3-soft); --color-chart-3-ink: var(--chart-3-ink);
  --color-chart-4: var(--chart-4); --color-chart-4-soft: var(--chart-4-soft); --color-chart-4-ink: var(--chart-4-ink);
  --color-chart-5: var(--chart-5); --color-chart-5-soft: var(--chart-5-soft); --color-chart-5-ink: var(--chart-5-ink);
  --color-chart-6: var(--chart-6); --color-chart-6-soft: var(--chart-6-soft); --color-chart-6-ink: var(--chart-6-ink);
  --color-chart-7: var(--chart-7); --color-chart-7-soft: var(--chart-7-soft); --color-chart-7-ink: var(--chart-7-ink);
}

@layer base {
  * { @apply border-border outline-ring/50; }
  body { @apply bg-background text-foreground font-sans antialiased; }
}
```

Суммы везде выводятся с `tabular-nums`. Формат: `Intl.NumberFormat('ru-RU')`, 0 знаков после запятой для целых и 2 для дробных, узкий пробел U+202F заменить на U+00A0 (в Onest его нет), затем ` ₽` — «24 350 ₽», «1 250 000,50 ₽». Даты: `date-fns` + локаль `ru` — `dd.MM.yyyy` в таблице, `d MMM` в мобильных карточках.

## 2. Категории

Семь категорий закреплены за `chart-1…7` в этом порядке. Порядок не меняется в фильтрах, Select и плитках; только диаграммы сортируются по сумме. Заливка (`chart-N`) — для полос, секторов и точек; бейдж и иконка — текст `chart-N-ink` на фоне `chart-N-soft`.

| Код | Название | Иконка lucide-react | Токен | Светлая: заливка / фон / текст | Тёмная: заливка / фон / текст | Контраст бейджа (св. / тём.) |
|---|---|---|---|---|---|---|
| `food` | Еда | `Utensils` | `chart-1` | #d9a11c / #faf2df / #846211 | #f5cf4f / #434126 / #f5cf4f | 5,0 / 6,9 |
| `transport` | Транспорт | `Car` | `chart-2` | #3f5fbd / #e4e9f6 / #3d5cb7 | #4a85dd / #213243 / #75a2e5 | 5,1 / 5,0 |
| `housing` | Жильё | `House` | `chart-3` | #2f9a80 / #e2f1ed / #22705d | #2bbb93 / #1b3d34 / #31bd96 | 5,1 / 5,0 |
| `entertainment` | Развлечения | `Ticket` | `chart-4` | #8b4d86 / #efe6ee / #884b83 | #c49af5 / #3a3647 / #c49af5 | 5,1 / 5,2 |
| `health` | Здоровье | `HeartPulse` | `chart-5` | #b8461b / #f5e5df / #a74019 | #ee8a2a / #42331f / #ef8f33 | 5,1 / 5,0 |
| `shopping` | Покупки | `ShoppingBag` | `chart-6` | #4fb4e8 / #e6f4fc / #2f6c8b | #ace3f1 / #354547 / #ace3f1 | 5,1 / 7,2 |
| `other` | Другое | `Shapes` | `chart-7` | #4a4044 / #e6e4e5 / #4a4044 | #7a6c6e / #2b2d2c / #a39a9b | 7,9 / 5,1 |

Карта в коде — один объект на всё приложение; классы пишутся целиком, иначе Tailwind их не увидит:

```tsx
import { Utensils, Car, House, Ticket, HeartPulse, ShoppingBag, Shapes, type LucideIcon } from "lucide-react";

export const CATEGORIES = {
  food:          { label: "Еда",        icon: Utensils,    chart: "var(--chart-1)", badge: "bg-chart-1-soft text-chart-1-ink", fill: "bg-chart-1" },
  transport:     { label: "Транспорт",  icon: Car,         chart: "var(--chart-2)", badge: "bg-chart-2-soft text-chart-2-ink", fill: "bg-chart-2" },
  housing:       { label: "Жильё",      icon: House,       chart: "var(--chart-3)", badge: "bg-chart-3-soft text-chart-3-ink", fill: "bg-chart-3" },
  entertainment: { label: "Развлечения", icon: Ticket,      chart: "var(--chart-4)", badge: "bg-chart-4-soft text-chart-4-ink", fill: "bg-chart-4" },
  health:        { label: "Здоровье",   icon: HeartPulse,  chart: "var(--chart-5)", badge: "bg-chart-5-soft text-chart-5-ink", fill: "bg-chart-5" },
  shopping:      { label: "Покупки",    icon: ShoppingBag, chart: "var(--chart-6)", badge: "bg-chart-6-soft text-chart-6-ink", fill: "bg-chart-6" },
  other:         { label: "Другое",     icon: Shapes,      chart: "var(--chart-7)", badge: "bg-chart-7-soft text-chart-7-ink", fill: "bg-chart-7" },
} satisfies Record<string, { label: string; icon: LucideIcon; chart: string; badge: string; fill: string }>;

export type CategoryCode = keyof typeof CATEGORIES;
```

- Цвет нигде не единственный носитель смысла: рядом всегда иконка или название. В светлой теме заливки `food` и `shopping` дают 2,3:1 к белому, поэтому сектора разделяются обводкой 2px `var(--card)`, а у полосы долей всегда есть подписи.
- Различимость при нарушениях цветовосприятия (протанопия, дейтеранопия, тританопия): минимум 9 ΔE в светлой теме, 10 ΔE в тёмной.
- Ключ «Развлечений» — `entertainment`; в README дизайн-системы он пока записан как `fun`.

## 3. Экраны и состояния

В приложении два маршрута (главная и форма траты) и 18 артбордов на холсте. Имя артборда = имя файла на холсте.

| # | Экран · состояние | Артборд | Ширина | Что показывает |
|---|---|---|---|---|
| 1 | Главная · данные | `Main` | 1440 | Фильтры, сводка по всем 37 тратам фильтра, «По категориям», таблица 10 строк, пагинация |
| 2 | Главная · загрузка | `MainLoading` | 1440 | Skeleton карточек, 7 строк категорий, 6 строк таблицы; фильтры доступны |
| 3 | Главная · пусто по фильтру | `MainEmpty` | 1440 | 0 ₽, пустая диаграмма, «За выбранный период трат нет» + «Сбросить фильтры», «Добавить трату» |
| 4 | Главная · ошибка загрузки | `MainError` | 1440 | «—» в карточках, «Не удалось загрузить траты» + «Повторить» |
| 5 | Главная · тёмная тема | `MainDark` | 1440 | То же, что 1, в `.dark` |
| 6 | Select «Категория» · открыт | `CategorySelect` | 360 | Список «Все» + 7 категорий с цветной точкой, hover |
| 7 | Диалог «Трата» · компонент | `ExpenseDialog` | 520 | Источник для 8–11: режимы new / errors / edit, светлая и тёмная |
| 8 | Новая трата · пустая форма | `DialogNew` | 1440 | Диалог поверх затемнённой главной; член семьи = текущий пользователь, дата = сегодня |
| 9 | Новая трата · ошибки валидации | `DialogErrors` | 1440 | «Укажите сумму», «Выберите категорию» |
| 10 | Редактирование · сохранение | `DialogEdit` | 1440 | Заполненная трата, «Сохраняем…» со спиннером, «Удалить» слева |
| 11 | Редактирование · тёмная тема | `DialogEditDark` | 1440 | То же, что 10, в `.dark`; затемнение чёрным 60% |
| 12 | Удаление · подтверждение | `DeleteConfirm` | 1440 | «Удалить трату?», сама трата, «Это действие нельзя отменить»; фокус на «Отмене» |
| 13 | Уведомления | `Toasts` | 1440 | «Трата добавлена», «Изменения сохранены», «Трата удалена», ошибка с «Повторить» |
| 14 | Мобильная главная · первый экран | `MobileMain` | 390 | «Фильтры (2)», чипы периодов, Итого + полоса долей, карточки, кнопка «+» |
| 15 | Мобильная главная · целиком | `MobileMainFull` | 390 | Раскрытая легенда, открытое меню «⋯», «Показать ещё 8» |
| 16 | Мобильные фильтры | `MobileFilters` | 390 | Нижняя шторка: член семьи, период, категория, «Применить» |
| 17 | Мобильная новая трата | `MobileCreate` | 390 | Форма во весь экран, сумма первой |
| 18 | Длинные значения | `EdgeCases` | 1440 | Комментарий 200 символов + Tooltip, 1 250 000,50 ₽, длинное имя — десктоп и мобильный |

- Решить: удаление с окном подтверждения (как сейчас) или мгновенное с «Отменить» в уведомлении

## 4. Компоненты по блокам

Все блоки собираются из 28 стандартных компонентов shadcn/ui; своё — только `className` с токенами. Высота элементов управления: `h-11 md:h-9` (44px на телефоне, 36px на десктопе). Фокус — стандартный `focus-visible:ring` shadcn, его не отключать.

```bash
npx shadcn@latest add button card table dialog alert-dialog select popover calendar input textarea badge skeleton sonner chart pagination dropdown-menu tooltip toggle-group avatar drawer collapsible progress input-group spinner empty item field label
```

### Шапка и фильтры

| Блок | Компонент | variant · size | Детали |
|---|---|---|---|
| Тема | Button | `outline` · `icon` (телефон: `ghost` · `icon`, `size-11`) | `Moon` / `Sun`, `aria-label="Тёмная тема"`; next-themes |
| «Добавить трату» | Button | `default` · `default` | Иконка `Plus`; на телефоне — плавающая: `size="icon"` + `fixed bottom-6 right-4 size-14 rounded-full shadow-lg` |
| Член семьи | Select + Avatar | SelectTrigger `w-[200px] max-w-[260px]` | «Все члены семьи» + имена; SelectValue сам обрезает длинное имя |
| Период | Popover + Calendar | Button `outline` · `default` | Calendar `mode="range"`, `numberOfMonths={2}`, `locale={ru}`; текст «1–31 окт 2026» |
| Быстрые периоды (десктоп) | ToggleGroup | `type="single"` · `default` · `sm` | Обёртка `bg-muted p-[3px] rounded-lg`; выбранный: `data-[state=on]:bg-card data-[state=on]:shadow-sm` |
| Категория | Select | SelectTrigger `w-[200px]` | В SelectItem точка `size-2.5 rounded-full` цвета `CATEGORIES[code].fill` |
| «Сбросить» | Button | `ghost` · `default` | Иконка `RotateCcw`; рендерится, только когда фильтры отличаются от стандартных |
| «Фильтры (2)» (телефон) | Button + Badge | Button `outline`, `rounded-full h-11`; Badge `default` | Иконка `SlidersHorizontal`; счётчик = число нестандартных фильтров в шторке |
| Чипы периодов (телефон) | ToggleGroup | `type="single"` · `outline` | Строка `overflow-x-auto`; элемент `h-11 rounded-full`; выбранный: `data-[state=on]:bg-foreground data-[state=on]:text-background` |
| Шторка фильтров (телефон) | Drawer | DrawerContent `rounded-t-xl` | Внутри три ToggleGroup (член семьи с Avatar, период, категория с точкой) + Button `outline` «Выбрать даты»; в DrawerHeader Button `link` «Сбросить»; в DrawerFooter Button `default` · `lg`, `w-full` «Применить» |

### Сводка

| Блок | Компонент | variant · size | Детали |
|---|---|---|---|
| Итого, Трат, Средняя трата | Card | — | CardDescription — подпись; итог `text-[40px] font-semibold tabular-nums tracking-tight` (телефон `text-4xl`), остальные `text-[28px]` |
| Строка «Сводка по всем N тратам по фильтру…» | текст | — | `text-muted-foreground`, `aria-live="polite"`; итоги считаются по всему фильтру, не по странице |
| «По категориям» (десктоп) | Card + Progress | Progress `h-2`, индикатор `bg-chart-N` | Строка: иконка в круге 28px + название · полоса (длина относительно максимума) · сумма 112px · доля 40px; сетка `grid-cols-[148px_1fr_112px_40px]`, сортировка по сумме |
| Полоса долей (телефон) | Chart (BarChart) | `layout="vertical"`, серии с одним `stackId` | Высота 12px, `radius` по краям, обводка 2px `var(--card)`; под ней всегда три главные доли текстом |
| Легенда (телефон) | Collapsible | CollapsibleTrigger как Button `ghost`, `w-full h-11` | «По категориям: 6» / «Скрыть категории», `ChevronDown` поворачивается |

### Таблица и список

| Блок | Компонент | variant · size | Детали |
|---|---|---|---|
| Таблица | Table (Data Table, TanStack) | `table-fixed` | Колонки: Дата 140 · Кто 176 · Категория 176 · Сумма 160 · Комментарий (остаток) · действия 104 px; строка 52px; шапка `bg-muted`; наведение `hover:bg-accent` |
| Сортировка Дата, Сумма | Button в TableHead | `ghost` · `sm` | `ArrowDown` / `ArrowUp` у активной, `ArrowUpDown` у неактивной; `aria-sort` на `th`; по умолчанию дата ↓ |
| Кто | Avatar | `size-7`, AvatarFallback `bg-secondary text-xs` | Имя `truncate`, полное имя — в `title` |
| Категория | Badge | `secondary` + `border-0 rounded-sm h-[22px] gap-1` + `CATEGORIES[code].badge` | Иконка `size-3.5` + название |
| Сумма | TableCell | — | `text-right font-medium tabular-nums whitespace-nowrap`; никогда не окрашивать |
| Комментарий | Tooltip | TooltipContent `max-w-[360px]` | Ячейка `truncate`; Tooltip только если текст обрезан (`scrollWidth > clientWidth`); триггер фокусируется (`tabIndex={0}`) |
| Действия в строке | Button × 2 | `ghost` · `icon` (`size-8`) | `Pencil`, `Trash2`; `text-muted-foreground`, при наведении на строку или фокусе `text-foreground`, корзина `hover:text-destructive`; `aria-label="Удалить: Кофе, 350 ₽"` |
| Пагинация | Pagination | PaginationLink `isActive` → `bg-primary text-primary-foreground` | Слева «Показано 1–10 из 37»; 10 строк на страницу |
| Карточка траты (телефон) | Item | `outline` | ItemMedia — иконка в круге 40px; ItemContent — ItemTitle `truncate` (комментарий) + ItemDescription `truncate` («Мадина · 30 сен»); ItemActions — сумма `shrink-0` + меню |
| Меню «⋯» (телефон) | DropdownMenu | триггер Button `ghost` · `icon`; пункты `h-11` | «Изменить» — DropdownMenuItem; «Удалить» — DropdownMenuItem `variant="destructive"` |
| «Показать ещё 8» (телефон) | Button | `outline` · `lg`, `w-full` | Заменяет Pagination; догружает по 10 |

### Состояния списка

| Состояние | Компонент | variant · size | Детали |
|---|---|---|---|
| Загрузка | Skeleton | — | Повторяет форму контента; заголовки таблицы настоящие; обёртка `aria-busy="true"`; пагинация скрыта |
| Пусто по фильтру | Empty | EmptyMedia `variant="icon"` (`Receipt`) | EmptyTitle «За выбранный период трат нет»; EmptyContent: Button `outline` «Сбросить фильтры» + Button `default` «Добавить трату» |
| Ошибка загрузки | Empty | EmptyMedia `variant="icon"` (`CloudOff`, `text-destructive`) | `role="alert"`; Button `outline` «Повторить» с `RefreshCw`; в карточках сводки «—» |

### Диалоги и уведомления

| Блок | Компонент | variant · size | Детали |
|---|---|---|---|
| Диалог «Новая трата» / «Редактирование траты» | Dialog | DialogContent `sm:max-w-[520px] rounded-xl` | DialogTitle `text-xl`; DialogDescription: «Траты из Telegram-бота появляются здесь сами…» / «Записано через Telegram-бота» |
| Поля формы | Field + Label | — | react-hook-form + zod; ошибка — FieldError с `AlertCircle`, на поле `aria-invalid`; необязательное поле помечено «— необязательно», звёздочек нет |
| Член семьи | Select + Avatar | SelectTrigger `w-full` | По умолчанию текущий пользователь «… (вы)» |
| Сумма, ₽ | InputGroup | InputGroupInput `h-14 text-[28px] font-semibold tabular-nums`, `inputMode="decimal"`; InputGroupAddon `align="inline-end"` «₽» | Принимает «,» и «.», до 2 знаков после запятой, > 0; подсказка «Можно с копейками: 349,90»; ошибка «Укажите сумму» |
| Категория | ToggleGroup | `type="single"` · `outline`, `grid grid-cols-4 gap-2` | Плитка `h-[76px] flex-col rounded-lg`; выбранная: фон `chart-N-soft`, рамка 2px `chart-N-ink` + `Check` в углу; ошибка «Выберите категорию», рамки `border-destructive` |
| Дата | Popover + Calendar | Button `outline`, `w-full justify-between`; Calendar `mode="single"` | По умолчанию «Сегодня, 16.10.2026»; вчера — «Вчера, …» |
| Комментарий | Textarea | `rows={2}`, `text-base` | Placeholder «Например: кофе с коллегами»; `maxLength={500}` |
| Кнопки диалога | Button | «Отмена» `secondary`; «Сохранить» `default` `min-w-[136px]`; «Удалить» (редактирование) `ghost` + `text-destructive`, слева | Сохранение: `disabled` + Spinner + «Сохраняем…», без снижения прозрачности (`disabled:opacity-100`) |
| Подтверждение удаления | AlertDialog | AlertDialogCancel `buttonVariants({variant:"secondary"})`; AlertDialogAction `buttonVariants({variant:"destructive"})` | Начальный фокус на «Отмене»; трата в блоке `bg-muted rounded-lg p-3`: иконка, комментарий, «Категория · Кто · дата», сумма |
| Уведомления | Sonner (Toaster) | `position="bottom-right"` (телефон `top-center`) | `toast.success("Трата добавлена", { description: "Кофе · 350 ₽ · Алишер" })`, 4 с; `toast.error("Не удалось сохранить", { description: "Попробуйте ещё раз…", action: { label: "Повторить" }, duration: Infinity, closeButton: true })` |

## 5. Брейкпоинты и адаптив

Главная граница — `md` (768px): ниже таблица становится карточками, фильтры уходят в нижнюю шторку, форма открывается во весь экран. Брейкпоинты — стандартные Tailwind, своих нет.

| Что | < 768 (база, `sm`) | 768–1023 (`md`) | 1024–1279 (`lg`) | ≥ 1280 (`xl`) |
|---|---|---|---|---|
| Поля страницы | `px-4` | `px-6` | `px-8` | `px-8`, контент `max-w-[1280px] mx-auto` |
| Шапка | Название `text-xl` + тема; шапка и строка фильтров `sticky top-0`; плавающая «+» | Название `text-2xl` + тема + «Добавить трату» | как `md` | как `md` |
| Фильтры | «Фильтры (N)» + чипы периодов в одной строке с горизонтальной прокруткой; остальное — в Drawer | В строку с переносом (`flex flex-wrap gap-3`) | В строку, может перенестись «Категория» | Все в одну строку |
| Сводка | Одна Card: Итого `text-4xl`, «14 трат · в среднем …», полоса долей + Collapsible-легенда | Три Card в ряд (`grid-cols-3`), ниже «По категориям» на всю ширину | Две колонки `grid-cols-[minmax(380px,5fr)_7fr] gap-6`: слева Итого + две Card, справа «По категориям» | как `lg` |
| Траты | Item-карточки (`md:hidden`), меню «⋯», «Показать ещё» | Data Table (`hidden md:block`) в `overflow-x-auto`, `min-w-[900px]` | Data Table, та же сетка колонок | Data Table без прокрутки |
| Форма траты | Во весь экран: DialogContent `max-md:h-dvh max-md:max-w-none max-md:rounded-none`, кнопки закреплены внизу, сумма первым полем с `autoFocus` | Dialog 520px по центру, порядок полей как в макете | как `md` | как `md` |
| AlertDialog | Ширина `calc(100%-2rem)`, кнопки в колонку, «Удалить» сверху | 440px, кнопки справа | как `md` | как `md` |
| Уведомления | `top-center`, чтобы не закрывать «+» | `bottom-right` | `bottom-right` | `bottom-right` |

- Размеры для касания до `md`: все кнопки, чипы, пункты меню и поля ≥ 44px; ввод — 16px (иначе iOS увеличивает страницу).
- У списка на телефоне нижний отступ `pb-28`, чтобы кнопка «+» не закрывала последнюю карточку.
- Длинные значения на всех ширинах: комментарий и имя обрезаются с многоточием (полный текст — Tooltip или `title`), сумма не переносится и не сжимается (`whitespace-nowrap shrink-0`); проверено на 1 250 000,50 ₽ и «Александра-Виктория Константинопольская» (артборд `EdgeCases`).
- Ширина контента 1280px шире токена `content-max` (1120px) в дизайн-системе: при 1120 фильтры не помещаются в одну строку. Для реализации ориентир — 1280px.