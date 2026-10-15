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
