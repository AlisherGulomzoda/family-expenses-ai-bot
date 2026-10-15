# BE-06. REST-сервер `cmd/api` по контракту

**Оценка:** 3 ч · **Зависит от:** BE-03, BE-05 · **Ветка:** `feature/BE-06-rest-api`

## Цель
Рабочий REST API для веб-панели, строго соответствующий `openapi.yaml`.

## Объём
- `internal/httpapi`:
  - роутинг на стандартном `http.ServeMux` (`"GET /api/expenses/{id}"`);
  - хендлеры по всем эндпоинтам BE-03, DTO ↔ домен (деньги через `expense.Money`, `json.Number` без float);
  - парсинг и валидация query-параметров (неверная дата/категория/sort → `400`/`422` с `fields`);
  - маппинг ошибок домена: `ErrNotFound` → 404, `ValidationError` → 422, прочее → 500 без утечки деталей (детали — в лог);
  - middleware: request ID, access-лог через slog (метод, путь, статус, длительность), recover от паники, лимит размера тела запроса;
  - `/healthz` (ping БД).
- `cmd/api/main.go`: конфиг → пул БД → репозитории → сервисы → сервер с таймаутами (`ReadHeaderTimeout` и др.), graceful shutdown.
- Сервис `api` в `docker-compose.yml`: backend-образ, команда `api`, `depends_on: migrate (service_completed_successfully)`, healthcheck на `/healthz`. Порт наружу не публикуется (доступ через `web`), для разработки — `ports` в `docker-compose.override.yml` или документированный флаг.

## Критерии приёмки
- [ ] Тесты хендлеров через `httptest` с fake-сервисом: успех, 400, 404, 422 для каждого эндпоинта.
- [ ] Ответы соответствуют схеме из `openapi.yaml` (проверка ключевых полей в тестах; опционально — валидация ответов через `kin-openapi` в тестах).
- [ ] `amount: 350.5` сохраняется как `35050` копеек и возвращается как `350.5`.
- [ ] Ручная проверка: `docker compose up postgres migrate api` + `curl` по всем эндпоинтам (команды — в README).

## Коммит
`feat(api): implement REST API for expenses web panel`

## Итог реализации (2026-10-16) — ✅ готово

Проверено:
- `go test -race` — httpapi покрыт на 95.4%; **каждый ответ (и каждый корректный запрос) в тестах валидируется по `api/openapi.yaml`** через `pb33f/libopenapi-validator` (поддерживает OpenAPI 3.1). Валидатор проверен на заведомо сломанных ответах — ловит.
- Полный прогон с Postgres (`-tags integration`) — зелёный; `golangci-lint` — 0.
- Вручную в compose: `api` → healthy; `curl` по всем эндпоинтам на реальной БД (create 201 + Location, list с сортировкой, summary, put с переносом и очисткой комментария, 400/404/422, delete 204); кириллица в комментарии сохраняется и ищется `ILIKE` без учёта регистра.

Решения:
- Хендлеры тестируются на **настоящем `expense.Service` поверх `memstore`** (а не на фейке) + отдельный «сломанный» сервис для 500 и паники.
- **400 vs 422** как в контракте: ошибки query/path и невалидный JSON/лишние поля → 400; отсутствующие обязательные поля, сумма строкой, >2 знаков, плохая дата, значения вне правил домена, неизвестный `user_id` → 422 с `fields`. Ошибки валидации домена из query (`limit`, `from > to`, `sort`) → 400.
- `PUT` — полная замена; отсутствующий/`null` comment очищает комментарий; `source` не меняется.
- Middleware: `X-Request-ID` (берётся от nginx или генерируется), access-лог slog, recover (500 без деталей, стек — в лог), `MaxBytesReader` 64 КБ → 413.
- Неизвестный путь под `/api/` → JSON 404.
- **`api healthcheck`** — подкоманда бинарника для Docker healthcheck (в distroless нет curl).
- Сервер: таймауты `ReadHeader 5s / Read 15s / Write 30s / Idle 60s`, graceful shutdown 10s.
- Compose: сервис `api`, `depends_on: migrate (completed_successfully)`, порт **`127.0.0.1:${API_PORT:-8081}`** (8080 оставлен для `web`), healthcheck.

Отличия от плана: нет.

Наблюдение для README: в Git Bash на Windows кириллица в `curl -d '...'` превращается в `?` (кодировка консоли) — в примерах использовать JSON-escape `\u…` или файл.
