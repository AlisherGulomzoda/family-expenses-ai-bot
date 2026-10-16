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

## Итог реализации (2026-10-16) — ✅ готово

Проверено (в `node:lts-alpine`): `format:check` ✓, `typecheck` ✓, `lint` — 0, `test` — **50/50** (деньги 24, даты 12, категории 3, хуки 9 поверх MSW, App 2), `build` ✓; MSW **не попадает** в production-бандл (ветка моков вырезается, размер бандла не изменился).

Что сделано:
- `npm run gen:api` → `src/api/schema.d.ts` (785 строк, коммитится); `src/api/client.ts` — `openapi-fetch`, `ApiError {status, code, message, fields}` (+ `cause` при сетевом сбое), `unwrap`.
- `src/api/hooks.ts` — `queryKeys` (всё под `['expenses']`), `useUsers`, `useExpenses` (`keepPreviousData`, 10 на страницу), `useExpensesInfinite` («Показать ещё»), `useSummary`, `useCreate/Update/DeleteExpense` (инвалидируют списки и сводку).
- `src/mocks/` — MSW 3: детерминированные ~40 трат 4 членов семьи за 45 дней относительно «сегодня»; обработчики повторяют семантику API (включительные периоды, сортировка с `id`, пагинация, сводка по всему фильтру, 400/404/422 с русскими текстами); `VITE_API_MOCKS=true` — моки в браузере; в тестах `server.listen({ onUnhandledFrame: 'error' })` — любой незамоканный запрос валит тест.
- `src/shared/money.ts` — `formatRub` («24 350 ₽», «1 250 000,50 ₽», U+202F → U+00A0), `parseRubInput` (запятая/точка/пробелы, ≤ 2 знаков, в копейках, без float); `dates.ts` — форматы дизайна, «Сегодня/Вчера, …», быстрые периоды (неделя Пн–Вс), `formatPeriod` («1–31 окт 2026»); `categories.ts` — `CATEGORIES` из дизайна, `satisfies Record<CategoryCode,…>` — переименование категории в API ломает сборку.

Найдено и исправлено по ходу:
- **`openapi-fetch` запоминает `fetch` при создании клиента** → MSW (и любые обёртки, поставленные позже) обходились, запросы шли в сеть (`ECONNREFUSED`). Клиенту передан `fetch: (r) => globalThis.fetch(r)`.
- `unwrap` скрывал причину сетевой ошибки — теперь она в `ApiError.cause`.
- `date-fns` сокращает месяцы словарно («сент.», «нояб.»), дизайн — «сен», «ноя» → своя таблица сокращений.

Отличия от плана:
- `openapi-typescript` 7.13 объявляет peer `typescript ^5`, в проекте TS 6 → `overrides` в `package.json` только для него (генерация проверена: те же 785 строк, что на TS 5 в BE-03).
- MSW 3: опция `onUnhandledRequest` переименована в `onUnhandledFrame`.
- `useCategories` не нужен: категории фиксированы и проверяются типами; эндпоинт `/api/categories` замокан для полноты.
