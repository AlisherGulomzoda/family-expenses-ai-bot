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
- `CreateInput`, `UpdateInput` (поля-указатели для частичного обновления), валидация: сумма > 0, категория валидна, дата не в будущем дальше чем на 1 день и не раньше 2000-01-01, комментарий ≤ 500 символов.
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
