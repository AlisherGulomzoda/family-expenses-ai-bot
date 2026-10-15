# FE-02. Слой API, моки и общие утилиты

**Оценка:** 2.5 ч · **Зависит от:** FE-01, BE-03 (контракт) · **Ветка:** `feature/FE-02-api-layer`
**Дизайн:** `design/design-spec.md` §1 (форматы сумм и дат), §2 (категории)

## Цель
Типобезопасный доступ к API и общие хелперы; разработка UI возможна **без бэкенда** (на моках).

## Объём
- `npm run gen:api` → `openapi-typescript ../backend/api/openapi.yaml -o src/api/schema.d.ts` (файл коммитится).
- `src/api/client.ts` — `openapi-fetch` с `baseUrl: "/api"`; единая обработка ошибок → `ApiError { status, code, message, fields? }`.
- `src/api/hooks.ts` (TanStack Query):
  - `useUsers`, `useExpenses(filters)` (`placeholderData: keepPreviousData` — таблица не мигает при смене страницы), `useExpensesInfinite(filters)` (для мобильного «Показать ещё»), `useSummary(filters)`;
  - `useCreateExpense`, `useUpdateExpense`, `useDeleteExpense` — после успеха инвалидируют `expenses` и `summary`.
  - Ключи запросов — из одного объекта `queryKeys`.
- **MSW**: обработчики на основе `examples` из OpenAPI + генератор ~40 реалистичных трат (4 члена семьи, октябрь 2026), фильтрация/сортировка/пагинация в памяти. Включается `VITE_API_MOCKS=true` в dev; используется и в тестах.
- `src/shared/`:
  - `money.ts` — `formatRub(amount)`: `Intl.NumberFormat('ru-RU')`, 0 знаков для целых и 2 для дробных, U+202F → U+00A0, суффикс « ₽»; `parseRubInput("349,90")` — запятая/точка, ≤ 2 знаков;
  - `dates.ts` — `date-fns` + `ru`: `dd.MM.yyyy`, `d MMM`, «Сегодня, …», «Вчера, …», диапазоны быстрых периодов (сегодня / неделя / этот месяц / прошлый месяц);
  - `categories.ts` — объект `CATEGORIES` из `design-spec.md` §2 (метка, иконка lucide, классы `badge`/`fill`, `chart`), порядок закреплён.

## Критерии приёмки
- [ ] Unit-тесты `formatRub` («24 350 ₽», «1 250 000,50 ₽», неразрывный пробел), `parseRubInput`, диапазонов периодов.
- [ ] `tsc` ловит несоответствие контракту (изменение поля в `schema.d.ts` ломает сборку).
- [ ] С `VITE_API_MOCKS=true` хуки возвращают данные без запущенного бэкенда.

## Коммит
`feat(frontend): add typed API client, query hooks, MSW mocks and shared formatters`
