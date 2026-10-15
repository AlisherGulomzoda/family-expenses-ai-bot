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

## Уточнения после BE-04 (повторить семантику `internal/memstore` 1:1)
- Сигнатуры — из `expense.Repository` (BE-04): `Update(ctx, id, owner, UpdateInput)`, `Delete` **возвращает** удалённую трату (`DELETE ... RETURNING`), `UserName` — через JOIN с `users`.
- `UpdateInput.Comment`: `nil` — не менять, `""` — NULL, иначе — значение; `updated_at = now()`.
- Неизвестный `user_id` (FK, и в Create, и в Update) → `&expense.ValidationError{Fields: {"user_id": "Неизвестный член семьи"}}`.
- `Summary.ByCategory`: `ORDER BY total DESC`, при равенстве — порядок категорий (`array_position` по списку кодов); пустой результат — `[]`, не `nil`.
- `UserRepo.List` — `ORDER BY name, telegram_id`. `Exists` не нужен.
- Хорошо бы общий набор контрактных тестов, который гоняется и против memstore, и против Postgres.

## Тесты (testcontainers-go, `postgres:17-alpine`, миграции из BE-02)
- [ ] CRUD-цикл траты.
- [ ] Каждый фильтр отдельно и в комбинации; период включительный по обеим границам.
- [ ] Сортировка по каждому полю в обе стороны; пагинация без дублей/пропусков.
- [ ] `Summary` совпадает с ручным подсчётом по тем же данным.
- [ ] **Изоляция**: Get/Update/Delete с `userID` пользователя A по трате B → `ErrNotFound`, данные B не изменились.
- [ ] `ChatHistoryRepo.Last` возвращает N последних в правильном порядке.
- Интеграционные тесты под build-тегом `integration` или пропускаются при недоступности Docker (`make test` — юнит, `make test-integration` — всё).

## Коммит
`feat(postgres): implement expense and user repositories with contract tests`

## Итог реализации (2026-10-16) — ✅ готово

Проверено: `make test-integration` (unit + Postgres 17 в testcontainers, `-race`) — 100 подтестов зелёные; `golangci-lint --build-tags integration` — 0 замечаний.

Что сделано:
- `sqlc.yaml`, запросы в `internal/postgres/queries/`, сгенерированный код в `internal/postgres/sqlcdb/` (коммитится, помечен `linguist-generated`). Команда — `make sqlc` (через Docker).
- `ExpenseRepo`, `UserRepo`, `Connect` (pgxpool + ping).
- **Изоляция в SQL**: `owner IS NULL OR user_id = owner` прямо в WHERE у Get/Update/Delete.
- **Сортировка без конкатенации SQL**: whitelist домена → статические `CASE` в `ORDER BY` + вторичная сортировка по `id`.
- **ILIKE-экранирование** `%`, `_`, `\` в поиске по комментарию; регистронезависимость для кириллицы проверена на `postgres:17-alpine`.
- **Контрактный набор тестов** `internal/expense/expensetest` (users, CRUD, изоляция владельца, неизвестный пользователь, фильтры, сортировка/пагинация, сводка) — гоняется и против `memstore`, и против Postgres. Нашёл и выровнял расхождение: при неизвестном новом владельце memstore отвечал `ErrNotFound`, Postgres — ошибкой валидации; теперь оба — `ValidationError{user_id}`.
- Тест миграций `up → reset → up` на реальной БД.

Отличия от плана:
- **`ChatHistoryRepo` перенесён в BE-09**: формат сообщений (роли, tool calls) определяет агент; репозиторий будет реализовывать интерфейс, объявленный в пакете агента. Таблица `chat_messages` уже есть (BE-02).
- sqlc не поддерживает `UPDATE` внутри CTE и `RETURNING` колонок из `DELETE ... USING` → `UpdateExpense` через `UPDATE ... FROM users`, `DeleteExpense` и `CreateExpense` через CTE. Неизвестный новый владелец в Update даёт 0 строк → различаем «не найдено» и «неизвестный пользователь» запросом `UserExists`.
- Строки разных запросов имеют одинаковые поля → конвертируются в один тип (`sqlcdb.GetExpenseRow(row)`), один маппер `toExpense`.
- Интеграционные тесты — под build-тегом `integration` (`go test ./...` не требует Docker).
