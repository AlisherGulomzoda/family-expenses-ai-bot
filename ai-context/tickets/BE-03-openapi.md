# BE-03. Контракт REST API — `openapi.yaml`

**Оценка:** 1.5 ч · **Зависит от:** BE-01 · **Ветка:** `feature/BE-03-openapi`

## Цель
Единый контракт между фронтом и бэком (contract-first). После этого тикета фронтенд генерирует типы и работает на моках, не дожидаясь реализации API.

## Объём
`backend/api/openapi.yaml` (OpenAPI 3.1):

| Метод | Путь | Ответ |
|---|---|---|
| GET | `/api/users` | `User[]` (`id` = telegram_id, `name`, `username?`) |
| GET | `/api/categories` | `Category[]` (`code`, `name`) |
| GET | `/api/expenses` | `{ items: Expense[], total: int }` |
| GET | `/api/expenses/summary` | `{ total: number, count: int, by_category: [{category, total, count}] }` |
| POST | `/api/expenses` | `201 Expense` |
| GET | `/api/expenses/{id}` | `Expense` |
| PUT | `/api/expenses/{id}` | `Expense` |
| DELETE | `/api/expenses/{id}` | `204` |
| GET | `/healthz` | `200` |

- Query-параметры списка: `user_id`, `from`, `to` (`YYYY-MM-DD`, включительно), `category`, `sort` (`spent_at|amount|created_at`, по умолчанию `spent_at`), `order` (`asc|desc`, по умолчанию `desc`), `limit` (1–100, по умолчанию 20), `offset`. Summary — те же фильтры без сортировки/пагинации.
- `Expense`: `id`, `user_id`, `user_name`, `amount` (рубли, number, ≤ 2 знаков), `category` (код), `spent_at` (date), `comment` (nullable), `source` (`bot` | `web`, readOnly — для подписи «Записано через Telegram-бота» в диалоге), `created_at`, `updated_at`.
- `ExpenseInput` (POST/PUT): `user_id`, `amount`, `category`, `spent_at`, `comment?`. Поля `source` во входе нет: POST из панели всегда `web`, PUT источник не меняет. PUT — полная замена (форма в UI всегда отправляет все поля).
- Ошибки: единая схема `Error { error: { code, message, fields? } }`; коды: `400 bad_request`, `404 not_found`, `422 validation_error`, `500 internal`.
- Примеры (`examples`) для основных ответов — пригодятся для MSW-моков на фронте.

## Критерии приёмки
- [ ] Спецификация валидна (`npx @redocly/cli lint` или аналог — команда в Makefile).
- [ ] Все поля и коды ошибок соответствуют B2 §4 и B3.
- [ ] `npx openapi-typescript backend/api/openapi.yaml` генерирует типы без ошибок.

## Коммит
`docs(api): add OpenAPI contract for expenses REST API`

## Итог реализации (2026-10-16) — ✅ готово

Проверено: `redocly lint` (2.64.0, ruleset `recommended`) — valid, 0 предупреждений; `openapi-typescript` 7.13 генерирует типы (785 строк) без ошибок, nullable и enum переносятся корректно. Команда — `make api-lint`.

Уточнения контракта (важны для BE-06 и FE):
- **400 vs 422**: 400 `bad_request` — невалидный JSON, query или path (плохая дата, неизвестная категория, `from > to`, нечисловой id), с `fields`; 422 `validation_error` — корректный JSON с недопустимыми значениями (сумма, неизвестный `user_id`, дата вне диапазона).
- `Money`: `> 0`, `≤ 100 000 000` ₽, ≤ 2 знаков — тот же лимит в домене (BE-04).
- `ExpenseInput`: `additionalProperties: false`, `comment` пустой/пробельный → `null`, `spent_at` не позже завтра и не раньше 2000-01-01.
- `POST` возвращает `201` + заголовок `Location`.
- `by_category` в summary — только категории с тратами, отсортированы по сумме по убыванию.
- `/healthz` → `200 {status: ok}` / `503 {status: unavailable}`.
- `message` в ошибках — по-русски, безопасно показывать в UI.
- `security: []` — явное «без авторизации»; правило `operation-4xx-response` отключено в `api/redocly.yaml` (у users/categories/healthz нет входных данных → нет честного 4xx).
