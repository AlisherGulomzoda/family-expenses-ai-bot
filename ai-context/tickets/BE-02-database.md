# BE-02. PostgreSQL, миграции, compose (postgres + migrate)

**Оценка:** 2 ч · **Зависит от:** BE-01 · **Ветка:** `feature/BE-02-database`

## Цель
Схема БД из B2 §3 и её автоматическое применение при `docker compose up`.

## Объём
- `backend/migrations/` (goose, SQL), встраиваются в бинарник через `embed`:
  - `00001_users.sql` — `users(telegram_id BIGINT PK, name TEXT NOT NULL, username TEXT, created_at)`.
  - `00002_expenses.sql` — `expenses` с `amount_kopecks BIGINT CHECK (> 0)`, `category TEXT CHECK (IN 7 кодов)`, `spent_at DATE`, `comment TEXT NULL`, `source TEXT NOT NULL CHECK (source IN ('bot','web'))` (без default — источник всегда задаёт приложение), `created_at`, `updated_at`, FK на `users`; индексы `(user_id, spent_at DESC)`, `(spent_at DESC)`.
  - `00003_chat_messages.sql` — `chat_messages(id, user_id FK, role TEXT, content JSONB, created_at)`, индекс `(user_id, id DESC)`.
  - Каждая миграция с `-- +goose Down`.
- `cmd/migrate`: применяет миграции (`up`), поддерживает `down`/`status` аргументом; выходит с кодом 0/1.
- `docker-compose.yml` (корень):
  - `postgres`: `postgres:17-alpine`, volume, healthcheck `pg_isready`, креды из `.env`.
  - `migrate`: backend-образ, команда `migrate up`, `depends_on: postgres (service_healthy)`, `restart: "no"`.
- `.env.example`: `POSTGRES_USER/PASSWORD/DB`, `DATABASE_URL` для контейнеров.

## Решения
- `updated_at` обновляет приложение (в UPDATE-запросе), а не триггер — проще читать и тестировать.
- Категории — `CHECK`, а не Postgres `ENUM`: добавление категории не требует `ALTER TYPE`.

## Критерии приёмки
- [ ] `docker compose up postgres migrate` → `migrate` завершается с кодом 0, таблицы созданы.
- [ ] Повторный запуск `migrate` — без ошибок (идемпотентно).
- [ ] `migrate down` откатывает все миграции.
- [ ] Вставка траты с `amount_kopecks <= 0`, неизвестной категорией или `source` вне `bot`/`web` отклоняется БД.

## Коммит
`feat(db): add PostgreSQL schema, goose migrations and compose services`
