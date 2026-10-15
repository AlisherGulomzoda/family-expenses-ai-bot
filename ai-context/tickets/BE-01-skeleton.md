# BE-01. Каркас Go-проекта, конфиг, логи, Dockerfile, Makefile

**Оценка:** 2 ч · **Зависит от:** — · **Ветка:** `feature/BE-01-skeleton`

## Цель
Пустой, но собирающийся и запускаемый проект со структурой из B2 §2, на которую ложатся все следующие тикеты.

## Объём
- `backend/go.mod` (модуль `github.com/<username>/family-expenses-ai-bot/backend`, актуальная стабильная Go).
- Директории из B2 §2: `cmd/{api,bot,migrate,eval}`, `internal/...`, `migrations/`, `eval/`, `api/`.
- `internal/config`: загрузка env через `caarlos0/env/v11`, **отдельная структура на каждый бинарник** (боту нужен токен, API — нет), обязательные поля валидируются при старте.

  | Переменная | Кто использует | По умолчанию |
  |---|---|---|
  | `DATABASE_URL` | api, bot, migrate, eval | — (обязательно) |
  | `HTTP_ADDR` | api | `:8080` |
  | `APP_TIMEZONE` | api, bot, eval | `Europe/Moscow` |
  | `LOG_LEVEL` | все | `info` |
  | `TELEGRAM_BOT_TOKEN` | bot | — (обязательно) |
  | `ALLOWED_TELEGRAM_IDS` | bot | пусто = все (с предупреждением в логе) |
  | `LLM_BASE_URL` | bot, eval | `https://api.deepseek.com` |
  | `LLM_API_KEY` | bot, eval | — (обязательно) |
  | `LLM_MODEL` | bot, eval | `deepseek-chat` |
  | `LLM_TIMEOUT` | bot, eval | `60s` |
  | `LLM_TEMPERATURE` | bot, eval | `0.1` |
  | `AGENT_HISTORY_LIMIT` | bot | `20` |
  | `AGENT_MAX_STEPS` | bot | `5` |

- `internal/logger` (или в `cmd`): `log/slog` JSON-хендлер, уровень из конфига.
- Каждый `cmd/*/main.go`: загрузка конфига, логгер, graceful shutdown по SIGINT/SIGTERM (`signal.NotifyContext`), пока без логики.
- `backend/Dockerfile`: multi-stage (`golang` → `gcr.io/distroless/static` или `alpine`), собирает все бинарники в один образ; команда выбирается в compose.
- `backend/Makefile`: `build`, `test`, `lint`, `sqlc`, `eval`, `run-api`, `run-bot`.
- `backend/.golangci.yml` с разумным набором линтеров (errcheck, govet, staticcheck, revive, gosec, …).
- В корне репозитория: `.env.example` со всеми переменными и комментариями, `.dockerignore`.

## Критерии приёмки
- [ ] `go build ./...` и `go vet ./...` без ошибок.
- [ ] `docker build` собирает образ; `docker run <img> api` стартует и корректно завершается по Ctrl+C.
- [ ] При отсутствии обязательной переменной бинарник падает с понятной ошибкой, какой именно переменной нет.
- [ ] Юнит-тест конфига: дефолты и ошибка обязательного поля.
- [ ] `.env` в `.gitignore`, `.env.example` в репо.

## Коммит
`chore(backend): scaffold Go project with config, logging and Dockerfile`
