# BE-04. Домен и сервис: траты, пользователи, деньги, изоляция по владельцу

**Оценка:** 3 ч · **Зависит от:** BE-01 · **Ветка:** `feature/BE-04-domain-service`

## Цель
Бизнес-логика без зависимостей от HTTP, Telegram, LLM и конкретной БД. Изоляция данных пользователя заложена **в типах**, а не в дисциплине вызывающего кода.

## Объём

### `internal/expense`
- `Expense` (ID, UserID, Amount, Category, SpentAt, Comment *string, Source, CreatedAt, UpdatedAt).
- `Source` — `bot` | `web`. **Задаётся сервисом, а не вызывающим кодом**: админский `Service.Create` → `web`, `UserScope.Create` (бот) → `bot`. Update источник не меняет.
- `Category` — тип-строка с 7 кодами (`food, transport, housing, entertainment, health, shopping, other`), `ParseCategory`, `Categories()` с русскими названиями.
- `Money` — `int64` копеек:
  - `ParseRubles(string) (Money, error)` — из строки/`json.Number` **без float** («350», «350.5», «350,50»); > 2 знаков после запятой и ≤ 0 — ошибка;
  - `Rubles() string` / `MarshalJSON` — рубли для API (`350.5`), `String()` — «350,50 ₽» для бота.
- `Filter` (UserID *int64, From/To *date, Category *Category, Query string, Sort, Order, Limit, Offset), `SortField` с whitelist (`spent_at`, `amount`, `created_at`).
- `Summary` (Total Money, Count int, ByCategory []CategoryTotal).
- `CreateInput`, `UpdateInput` (поля-указатели для частичного обновления), валидация: сумма > 0 и ≤ 100 000 000 ₽ (как `Money.maximum` в `openapi.yaml`), категория валидна, пустой/пробельный комментарий → `nil`, дата не в будущем дальше чем на 1 день и не раньше 2000-01-01, комментарий ≤ 500 символов.
- Ошибки: `ErrNotFound`, `ValidationError{Fields map[string]string}`.
- `Repository` — интерфейс (реализация в BE-05).
- `Service` — методы для **админа (веб-панель)**: `List`, `Summary`, `Get`, `Create`, `Update`, `Delete` — по всем пользователям.
- **`UserScope`** — `Service.ForUser(userID) UserScope`: те же операции, но `userID` зашит внутрь и **не принимается параметрами**. Все запросы идут в репозиторий с `user_id` (`... WHERE id=$1 AND user_id=$2`). Чужая трата → `ErrNotFound` (не «доступ запрещён» — не раскрываем существование).
- `Clock` интерфейс + `Location` (TZ из конфига) для «сегодня по умолчанию».

### `internal/user`
- `User` (TelegramID, Name, Username), `Repository` интерфейс, `Service.Upsert`, `Service.List`.

## Критерии приёмки
- [ ] Табличные юнит-тесты `ParseRubles` (валидные/невалидные, запятая/точка, копейки, переполнение).
- [ ] Юнит-тесты валидации `CreateInput`/`UpdateInput`.
- [ ] Тест: трата через `Service.Create` получает `source=web`, через `UserScope.Create` — `source=bot`; update не меняет `source`.
- [ ] Тест с fake-репозиторием: `UserScope` пользователя A не видит/не меняет/не удаляет трату B → `ErrNotFound`.
- [ ] Пакет `expense` не импортирует ничего из `httpapi`, `telegram`, `agent`, `llm`, `postgres`.

## Коммит
`feat(expense): add domain model, money type and user-scoped service`

## Итог реализации (2026-10-16) — ✅ готово

Проверено: `go test -race` — всё зелёное (покрытие: expense 88.5%, user 100%, memstore 73%, config 92%); `golangci-lint` — 0 замечаний.

Что получилось (важно для следующих тикетов):
- **Изоляция в типах**: `UserScope.Create(Draft)` — у `Draft` нет поля владельца; `UserScope.Update(Patch)` — `Patch` не умеет менять владельца (это умеет только админский `UpdateInput{UserID, Patch}`); `List/Summary` перезаписывают `Filter.UserID`; Get/Update/Delete передают `owner` в репозиторий. Чужая трата → `ErrNotFound`.
- **Интерфейс репозитория** (реализует BE-05): `Create(NewExpense)`, `Get/Update/Delete(id, owner *int64, ...)`, `Update` принимает `UpdateInput`, **`Delete` возвращает удалённую трату** (бот скажет, что удалил), `List/Count/Summary(Filter)`. `Filter` приходит уже нормализованным.
- **Даты** — `time.Time` в полночь UTC (как `DATE` в pgx); «сегодня» = `clock.Today(clock, loc)` в часовом поясе из конфига. Пакет `internal/clock` (`System`, `Fixed`).
- **Money**: парсинг «350», «349,90», «1 200» без float; `Rubles()` для JSON (`350.5`), `String()` для людей («1 200 ₽», неразрывные пробелы), JSON принимает только число.
- **Комментарий**: пробелы обрезаются; в `Draft` пустой → `nil`; в `Patch` пустой → `""` = «очистить» (репозиторий ставит NULL).
- **Тексты ошибок валидации** — по-русски, ключи = поля API / query-параметры.
- **`internal/memstore`** — in-memory `Users()` / `Expenses()` с семантикой Postgres; используется в тестах и пойдёт в eval (BE-11) и тесты агента (BE-08/09).
- Архитектурный тест `deps_test.go`: домен импортирует только stdlib и `internal/clock`.

Отличия от плана:
- `user.Repository` без `Exists` — не понадобился: неизвестный `user_id` ловит FK (в memstore — та же ошибка `ValidationError{user_id}`).
- `user.Service.Upsert` подставляет имя, если в профиле Telegram его нет (`@username` или «Пользователь <id>»).
- Найден и исправлен баг в memstore до коммита: `Get` отдавал указатель на хранимый комментарий — тест `TestReturnedValuesAreCopies`.
