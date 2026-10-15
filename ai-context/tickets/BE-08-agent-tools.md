# BE-08. Реестр тулов агента

**Оценка:** 2.5 ч · **Зависит от:** BE-04, BE-07 · **Ветка:** `feature/BE-08-agent-tools`

## Цель
Набор тулов, через которые LLM работает с тратами пользователя. Тулы не зависят от протокола (используются агентом и, опционально, MCP-сервером в BE-13).

## Объём
- `internal/agent/tools`:
  - `Tool{Name, Description, Schema json.RawMessage, Handler func(ctx, expense.UserScope, json.RawMessage) (any, error)}`, `Registry` (список, поиск по имени, `Defs() []llm.ToolDef`).
  - **Handler получает только `expense.UserScope`** — `user_id` нет ни в схеме, ни в аргументах.

| Тул | Аргументы (JSON Schema) | Результат |
|---|---|---|
| `add_expense` | `amount` (string, «350.50»), `category` (enum 7 кодов), `spent_at` (date, опц. → сегодня), `comment` (опц.) | созданная трата |
| `list_expenses` | `from`, `to` (date, опц.), `category` (опц.), `query` (опц., поиск по комментарию), `sort` (`spent_at`/`created_at`), `limit` (1–50, по умолч. 10) | `{items, total}` |
| `summarize_expenses` | `from`, `to` (date), `category` (опц.) | `{total, count, by_category}` |
| `update_expense` | `id` (int), `amount?`, `category?`, `spent_at?`, `comment?` | обновлённая трата |
| `delete_expense` | `id` (int) | удалённая трата (чтобы бот мог сказать, что именно удалил) |

  - Описания тулов и полей — подробные, с примерами (это часть промпт-инжиниринга: «используй `list_expenses` с `sort=created_at, limit=1`, чтобы найти последнюю трату»).
  - `source` в схемах тулов **отсутствует**: `add_expense` работает через `UserScope`, который сам ставит `source=bot`.
  - Сумма в аргументах — **строка**, парсится `expense.ParseRubles` (без float).
  - Результаты — компактный JSON: `id`, `amount` («350,00 ₽»), `category` (код + русское имя), `spent_at`, `comment`. Без лишних полей — экономим контекст.
  - Ошибки валидации/`ErrNotFound` возвращаются **как результат тула** (`{"error": "..."}`), а не роняют цикл — модель может исправиться или переспросить.

## Критерии приёмки
- [ ] Схемы всех тулов — валидный JSON Schema (тест).
- [ ] Юнит-тесты каждого handler на fake `UserScope`: успех, невалидные аргументы, чужой/несуществующий `id` → `{"error": "not found"}`.
- [ ] Тест: ни в одной схеме нет поля `user_id`.

## Коммит
`feat(agent): add tool registry for expense operations`
