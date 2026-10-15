# FE-05. Таблица трат: сортировка, пагинация, состояния (десктоп)

**Оценка:** 2.5 ч · **Зависит от:** FE-03 · **Ветка:** `feature/FE-05-table`
**Дизайн:** артборды `Main`, `MainLoading`, `MainEmpty`, `MainError`, `EdgeCases`; `design-spec.md` §4 «Таблица и список», «Состояния списка»

## Цель
Список трат по фильтру с серверной сортировкой и пагинацией.

## Объём
- Data Table (TanStack Table, `manualSorting`, `manualPagination`), `table-fixed`, колонки: Дата 140 · Кто 176 · Категория 176 · Сумма 160 · Комментарий (остаток) · действия 104 px; строка 52px; шапка `bg-muted`; `hover:bg-accent`.
- Сортировка «Дата» и «Сумма» — Button `ghost sm` в заголовке, иконки `ArrowDown`/`ArrowUp`/`ArrowUpDown`, `aria-sort`; состояние в URL (FE-03).
- Ячейки: Avatar + имя (`truncate`, `title`); Badge категории (`CATEGORIES[code].badge`, иконка); сумма `text-right tabular-nums whitespace-nowrap`; комментарий `truncate` + Tooltip **только если текст обрезан**, `tabIndex={0}`.
- Действия: `Pencil` (открывает диалог редактирования, FE-06), `Trash2` (подтверждение, FE-07); `aria-label="Удалить: Кофе, 350 ₽"`.
- Pagination: «Показано 1–10 из 37», 10 на страницу (`limit=10`), активная страница `bg-primary`.
- Состояния: Skeleton (настоящие заголовки, 6 строк, `aria-busy`), Empty `Receipt` («За выбранный период трат нет» + «Сбросить фильтры» + «Добавить трату»), Error `CloudOff` (`role="alert"`, «Повторить» → refetch).
- Обёртка `hidden md:block`, на `md` — `overflow-x-auto min-w-[900px]`.

## Критерии приёмки
- [ ] Тест (MSW): смена сортировки и страницы меняет запрос; пустое и ошибочное состояния отображаются.
- [ ] Длинный комментарий, сумма 1 250 000,50 ₽, длинное имя не ломают раскладку (артборд `EdgeCases`).
- [ ] Совпадает с артбордом `Main` в обеих темах.

## Коммит
`feat(frontend): add expenses data table with sorting, pagination and states`
