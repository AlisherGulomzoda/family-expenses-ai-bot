# BE-05. Репозиторий на pgx + sqlc, интеграционные тесты, тест изоляции

**Оценка:** 3 ч · **Зависит от:** BE-02, BE-04 · **Ветка:** `feature/BE-05-repository`

## Цель
Реализации `expense.Repository`, `user.Repository` и хранилища истории диалога на PostgreSQL, проверенные на настоящей БД.

## Объём
- `backend/sqlc.yaml`, запросы в `internal/postgres/queries/*.sql`, сгенерированный код коммитится.
- `internal/postgres`:
  - пул `pgxpool`, `New(ctx, databaseURL)` с ping;
  - `ExpenseRepo`: `Create`, `Get(id, userID *int64)`, `Update(id, userID *int64, patch)`, `Delete(id, userID *int64)`, `List(filter)`, `Count(filter)`, `Summary(filter)`.
    - Необязательные фильтры через `sqlc.narg` (`(@category::text IS NULL OR category = @category)`).
    - Поиск по комментарию — `comment ILIKE '%' || @query || '%'` (спецсимволы `%`/`_` экранируются).
    - Сортировка: whitelist из домена → `ORDER BY CASE ...` в sqlc; всегда вторичная сортировка `id DESC` для стабильной пагинации.
    - Update/Delete при заданном `userID` — `WHERE id=$1 AND user_id=$2`, 0 строк → `ErrNotFound`.
    - `Summary` — `SUM`/`COUNT` + `GROUP BY category` одним проходом по тем же фильтрам.
  - `UserRepo`: `Upsert` (`ON CONFLICT DO UPDATE` имени/username), `List`, `Exists`.
  - `ChatHistoryRepo`: `Append(userID, messages...)`, `Last(userID, n)` (в хронологическом порядке).
  - Маппинг ошибок pgx (`pgx.ErrNoRows` → `ErrNotFound`, FK violation по `user_id` → `ValidationError{"user_id": "неизвестный пользователь"}`).

## Тесты (testcontainers-go, `postgres:17-alpine`, миграции из BE-02)
- [ ] CRUD-цикл траты.
- [ ] Каждый фильтр отдельно и в комбинации; период включительный по обеим границам.
- [ ] Сортировка по каждому полю в обе стороны; пагинация без дублей/пропусков.
- [ ] `Summary` совпадает с ручным подсчётом по тем же данным.
- [ ] **Изоляция**: Get/Update/Delete с `userID` пользователя A по трате B → `ErrNotFound`, данные B не изменились.
- [ ] `ChatHistoryRepo.Last` возвращает N последних в правильном порядке.
- Интеграционные тесты под build-тегом `integration` или пропускаются при недоступности Docker (`make test` — юнит, `make test-integration` — всё).

## Коммит
`feat(postgres): implement expense, user and chat history repositories`
