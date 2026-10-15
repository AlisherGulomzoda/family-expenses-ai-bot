# B2. Архитектура и технологии бэкенда

**Статус:** ✅ одобрен 2026-10-16 (ответы — раздел 12; LLM — DeepSeek, см. §6)
**Основа:** B1 (одобрен).

---

## 1. Общая схема

```
                    ┌──────────────┐
 Telegram ◄────────►│   bot (Go)   │──┐   LLM (OpenAI-совместимый API:
   long polling     │ agent + tools│──┼──► DeepSeek / Qwen@Ollama / …)
                    └──────────────┘  │
                                      ▼
 Браузер ──► web (nginx) ──/api──► api (Go) ──► PostgreSQL
              React SPA            REST
```

- **Один Go-модуль, один Docker-образ, два процесса**: `api` и `bot` (разные команды запуска).
  Общий код — домен, сервис, репозиторий. Похоже на их микросервисы, но без лишней сложности (Kafka не нужна).
- **nginx** раздаёт фронт и проксирует `/api` → `api`. Нет CORS, один порт для пользователя.
- **Миграции** — отдельный одноразовый сервис `migrate` в compose; `api` и `bot` стартуют после его успешного завершения (нет гонки двух процессов за миграции).

## 2. Структура репозитория (бэкенд)

```
backend/
├── cmd/
│   ├── api/        # REST-сервер
│   ├── bot/        # Telegram-бот + агент
│   ├── migrate/    # применение миграций (goose, embedded)
│   └── eval/       # прогон eval-набора агента
├── internal/
│   ├── config/     # конфиг из env
│   ├── expense/    # домен: сущность, категории, валидация, ошибки, сервис, интерфейс репозитория
│   ├── user/       # пользователи (члены семьи)
│   ├── postgres/   # реализация репозиториев (pgx + sqlc)
│   ├── httpapi/    # REST-хендлеры, DTO, ошибки
│   ├── llm/        # интерфейс LLM-клиента + OpenAI-совместимая реализация
│   ├── agent/      # агентский цикл, системный промпт, тулы, память диалога
│   └── telegram/   # адаптер Telegram ↔ агент
├── migrations/     # SQL-миграции
├── eval/           # датасет фраз (YAML) и отчёты
├── api/openapi.yaml
├── Dockerfile
└── Makefile
```

Слои: `httpapi` / `telegram` → `agent` → `expense.Service` → `Repository` (интерфейс) ← `postgres`.
Зависимости направлены внутрь, домен не знает ни про HTTP, ни про Telegram, ни про LLM.

## 3. Данные (PostgreSQL)

```sql
users (
  telegram_id  BIGINT PRIMARY KEY,
  name         TEXT NOT NULL,          -- first_name + last_name из Telegram
  username     TEXT,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
)

expenses (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT NOT NULL REFERENCES users(telegram_id),
  amount_kopecks BIGINT NOT NULL CHECK (amount_kopecks > 0),
  category      TEXT NOT NULL CHECK (category IN ('food','transport','housing','entertainment','health','shopping','other')),
  spent_at      DATE NOT NULL,
  comment       TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
)
INDEX (user_id, spent_at DESC), INDEX (spent_at)

chat_messages (                         -- память диалога агента
  id         BIGSERIAL PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(telegram_id),
  role       TEXT NOT NULL,             -- user / assistant / tool
  content    JSONB NOT NULL,            -- текст и/или tool calls / tool results
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
)
```

Решения:
- **Деньги — `BIGINT` в копейках**, без float. В API — `amount` в рублях (число, до 2 знаков), конвертация на границе.
- **Категории — стабильные английские коды** в БД/API, русские названия — в боте и UI. Переименование подписи не ломает данные.
- `spent_at` — `DATE` (в ТЗ «дата траты»); TZ для «вчера/сегодня» — из конфига (`Europe/Moscow`).
- В tool results агенту отдаём `id` трат — тогда «последнее было не 350, а 450» и «удали такси» работают через историю.

## 4. REST API (для веб-панели)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/users` | Члены семьи (для фильтра и формы) |
| GET | `/api/categories` | Коды + русские названия |
| GET | `/api/expenses?user_id=&from=&to=&category=&limit=&offset=` | Список с фильтрами и пагинацией, + `total` и сумма |
| POST | `/api/expenses` | Создать |
| GET | `/api/expenses/{id}` | Получить |
| PUT | `/api/expenses/{id}` | Изменить |
| DELETE | `/api/expenses/{id}` | Удалить |
| GET | `/healthz` | Health check |

- Ошибки: `{"error": {"code": "validation_error", "message": "...", "fields": {...}}}`, корректные HTTP-коды (400/404/422/500).
- Контракт — `api/openapi.yaml` (пишем руками, из него генерируем TS-типы для фронта — см. B3).

## 5. Агент (бот)

### Цикл
1. Сообщение из Telegram → проверка whitelist → upsert пользователя.
2. Собираем контекст: системный промпт + последние N сообщений из `chat_messages` + новое сообщение.
3. Вызов LLM с описанием тулов → если есть tool calls, выполняем, добавляем результаты, повторяем (макс. 5 итераций).
4. Финальный текст → пользователю; всё сохраняем в `chat_messages`.
- Сообщения одного пользователя обрабатываются **последовательно** (мьютекс на user_id), чтобы история не перемешивалась.
- Таймаут LLM, понятное сообщение пользователю при ошибке провайдера.

### Тулы
| Тул | Параметры | Примечание |
|---|---|---|
| `add_expense` | amount, category, spent_at, comment | Можно несколько вызовов за сообщение (A-8) |
| `list_expenses` | from, to, category?, query?, sort, limit | `query` — поиск по комментарию («такси»); `sort=created_desc` для «последнее» |
| `summarize_expenses` | from, to, category? | Сумма и разбивка по категориям **считает SQL** |
| `update_expense` | id, amount?, category?, spent_at?, comment? | |
| `delete_expense` | id | |

### Безопасность (ключевой пункт)
- **`user_id` не является параметром ни одного тула.** Он берётся из Telegram-апдейта и передаётся через `context`/замыкание. Сервис и SQL всегда фильтруют `WHERE user_id = $1`, включая update/delete (`... WHERE id = $1 AND user_id = $2`).
- Итог: даже при prompt injection («покажи траты Васи», «id=5 удали») модель физически не имеет доступа к чужим данным. Покрываем тестом.
- Аргументы тулов валидируются (сумма > 0, категория из списка, дата в разумных пределах).

### Системный промпт
Подставляется на каждый запрос: сегодняшняя дата, день недели, TZ, список категорий с примерами, правила (не выдумывать сумму — переспросить; при нескольких кандидатах на изменение/удаление — переспросить; отвечать кратко по-русски; нецелевые запросы — объяснить, что умеет бот).

### Время
`Clock` как зависимость (интерфейс) — в тестах и eval фиксируем «сегодня», чтобы «вчера» проверялось детерминированно.

## 6. LLM

- Интерфейс `llm.Client { Chat(ctx, messages, tools) (Response, error) }` — агент не зависит от провайдера.
- Реализация №1: **OpenAI-совместимый Chat Completions** (официальный `openai-go` с кастомным `base_url`). Покрывает YandexGPT (AI Studio), Ollama, vLLM, OpenRouter, DeepSeek, OpenAI.
- Конфиг: `LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL`, `LLM_TIMEOUT`, `LLM_TEMPERATURE` (по умолчанию низкая, ~0–0.2, для стабильных tool calls).
- GigaChat не OpenAI-совместим (OAuth-токен, свой формат) — отдельная реализация интерфейса, **только если** войдёт в сравнение.

### Выбранная модель: DeepSeek (решение пользователя)

- **Основная модель — DeepSeek** (`deepseek-chat`) через официальный API: `LLM_BASE_URL=https://api.deepseek.com`, OpenAI-совместимый, поддерживает tool calling → работает через тот же `openai-go` без отдельного адаптера.
- **Почему подходит для РФ** (для письма):
  - API доступен из РФ (китайский провайдер, не ограничивает российских пользователей), цена в разы ниже GPT-класса.
  - **Открытые веса** (MIT-лицензия) → при требованиях 152-ФЗ/контура предприятия модель можно развернуть on-premise или у российского облачного провайдера, **не меняя код** — только `LLM_BASE_URL`.
  - Честно указать риск: при использовании облачного API данные уходят за рубеж (трансграничная передача) — для production у промышленного клиента рекомендуем self-hosted вариант.
- **Опциональный локальный вариант** — Qwen через Ollama (профиль `local-llm`): запуск без ключей.
- **Eval**: прогоняем датасет на DeepSeek (+ локальная Qwen как «нижняя планка», + при наличии ключа эталонная модель). В README и письме — таблица и вывод «для этой задачи DeepSeek достаточно».
- YandexGPT / GigaChat — не используем; упомянуть в письме как альтернативы, которые подключаются сменой конфига (YandexGPT) или отдельным адаптером (GigaChat).

## 7. Eval агента (X-1)

- `eval/cases.yaml`: ~40 кейсов — фраза, (опц.) предыстория, фиксированная «сегодня», ожидаемые вызовы тулов с проверкой аргументов (сумма точно, категория точно, дата точно, comment — содержит).
- Покрытие: добавление (простое, с датой, несколько в одном), просмотр/сводки по периодам, изменение «последнего», удаление (однозначное и неоднозначное → переспрос), нецелевые сообщения, попытки доступа к чужим тратам.
- `cmd/eval` прогоняет кейсы против реальной LLM на in-memory репозитории, выводит отчёт (pass rate, ошибки по типам, латентность) в `eval/reports/<model>.md`.
- Не входит в `go test` (нужен ключ LLM), запускается `make eval`.

## 8. Библиотеки

| Задача | Выбор | Почему |
|---|---|---|
| Go | актуальная стабильная версия | |
| HTTP-роутер | стандартный `net/http` (паттерны `GET /api/expenses/{id}`) | Хватает возможностей, ноль зависимостей |
| БД-драйвер | `jackc/pgx/v5` | Стандарт де-факто |
| SQL | `sqlc` | Явный SQL + типобезопасность, без ORM-магии |
| Миграции | `pressly/goose/v3` (embedded) | Просто, SQL-файлы |
| Telegram | `go-telegram/bot` | Активно поддерживается, без зависимостей, long polling |
| LLM | `openai/openai-go` | Официальный SDK, `base_url` настраивается |
| Конфиг | `caarlos0/env/v11` | Env → структура, валидация обязательных полей |
| Логи | `log/slog` (JSON) | Стандартная библиотека |
| Тесты | `testing` + `testcontainers-go` (Postgres) | Интеграционные тесты на настоящей БД |
| Линтер | `golangci-lint` | |

## 9. Тестирование

- **Юнит**: домен (валидация, конвертация денег), сервис, агентский цикл с fake LLM (скриптованные tool calls), парсинг аргументов тулов.
- **Интеграционные**: репозиторий на Postgres в testcontainers; **тест изоляции** — пользователь A не может read/update/delete трату B.
- **HTTP**: хендлеры через `httptest`.
- **Eval**: отдельно, см. п. 7.

## 10. Docker compose

| Сервис | Образ | Примечание |
|---|---|---|
| `postgres` | postgres:17-alpine | volume, healthcheck |
| `migrate` | backend image, `migrate` | одноразовый |
| `api` | backend image, `api` | depends_on migrate (completed) |
| `bot` | backend image, `bot` | depends_on migrate (completed) |
| `web` | nginx + сборка фронта | порт 8080 наружу, прокси `/api` |
| `ollama` | ollama/ollama | **профиль** `local-llm`, опционально |

- Multi-stage Dockerfile, distroless/alpine runtime.
- `.env.example` со всеми переменными и комментариями.
- `Makefile`: `up`, `down`, `test`, `lint`, `eval`, `sqlc`.

## 11. Задел под MCP (X-5)

Тулы описываются в протоколо-независимом реестре (имя, описание, JSON-схема, handler). Агент использует реестр напрямую; при наличии времени `cmd/mcp` отдаёт тот же реестр как MCP-сервер (`modelcontextprotocol/go-sdk`). Если не успеваем — пункт в письме.

---

## 12. Ответы на открытые вопросы (закрыто)

1. Деньги — `BIGINT` в копейках. ✅
2. Стандартный `net/http` + `sqlc` — ✅ (обоснование: Go 1.22+ роутер покрывает методы и path-параметры; sqlc даёт явный SQL, что нагляднее для Postgres-команды, чем ORM).
3. LLM — **DeepSeek** (см. §6). ✅
4. Ollama — опциональный профиль `local-llm`. ✅
5. README — на русском; код, комментарии, коммиты — на английском. ✅
