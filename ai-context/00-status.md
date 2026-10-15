# Статус работы

_Обновлено: 2026-10-16_

## Текущий этап

**Этап 3 — Реализация.** Готово: **BE-01…BE-06** ✅ (ядро бэкенда). Следующий тикет — **BE-07** (LLM-клиент, DeepSeek) — понадобится API-ключ DeepSeek для ручной проверки.

| Этап | Статус |
|---|---|
| Бэклоги B1–B3 (`backlog/`) | ✅ одобрены; дизайн получен, копия — `design/design-spec.md` |
| Тикеты BE-01…13, FE-01…10, CM-01 (`tickets/README.md`) | ✅ одобрены вместе с порядком реализации |
| Реализация | ▶️ BE-01…BE-05 ✅ (PR #1–#5), BE-06 ✅ — ждём коммит/пуш (C8) |

## Следующий шаг

1. Пользователь коммитит и пушит BE-06 (C8 в `commits.md`), сливает в `main`.
2. По сигналу «продолжай» — BE-07. Для ручной проверки нужен `LLM_API_KEY` DeepSeek в `.env` (сам ключ ассистенту не нужен).

## Локальный запуск инфраструктуры

- `docker compose up -d --build api` — БД + миграции + API на `http://127.0.0.1:8081` (`API_PORT`);
- `docker compose up -d postgres migrate` — только БД + миграции; `docker compose run --rm migrate migrate status|down|reset|up`.
- `docker compose down` — остановить (данные в volume `postgres-data`); `down -v` — удалить и данные.

## Как проверять бэкенд на Windows

Windows Smart App Control блокирует запуск свежесобранных тестовых `.exe` (`An Application Control policy has blocked this file`). Поэтому тесты и линтер гоняем в Docker:
- тесты: `docker run --rm -v "<путь>/backend:/src" -v gomodcache:/go/pkg/mod -w /src golang:1.27 go test -race -count=1 ./...` (Debian-образ: в alpine нет cgo для `-race`; volume `gomodcache` кэширует модули)
- интеграционные (Postgres в testcontainers): добавить `-v /var/run/docker.sock:/var/run/docker.sock -e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal` и `go test -tags integration` — см. цель `test-integration` в Makefile
- линтер: `docker run --rm -v "<путь>/backend:/src" -w /src golangci/golangci-lint:v2.14.0 golangci-lint run --build-tags integration ./...`
- sqlc: `docker run --rm -v "<путь>/backend:/src" -w /src sqlc/sqlc:1.31.1 generate`
- в Git Bash перед командой нужен `MSYS_NO_PATHCONV=1`. `make` на машине пользователя не установлен — Makefile для Linux/macOS и проверяющих.
- У пользователя `core.autocrlf=true`; в репо есть `.gitattributes` с `eol=lf`. Если `gofmt -l` вдруг показывает много файлов — проверить `git ls-files --eol` на `w/crlf`.

## Заметки

- LLM: DeepSeek (B2 §6). Ключ DeepSeek понадобится к BE-07 (ручная проверка) и BE-11 (eval). Цифры eval — в письмо.
- Окружение: Windows 11, Go 1.27.1, Docker Desktop.
- Если директорию переносят через `git clone`, папки `ai-context/private/` и файла `test-task-expenses.md` не будет (они в `.gitignore`). Суть ТЗ сохранена в `01-task.md` и `backlog/B1-requirements.md`.

## Этапы проекта

1. Бэклоги (B1 → B2 → B3) — ✅.
2. Декомпозиция на тикеты — ✅.
3. Реализация по тикетам (после каждого — коммит и пуш пользователем) — ▶️.
4. README, сопроводительное письмо (CM-01).

## Сроки

- ТЗ получено вечером 2026-10-16 (пт). Формальный дедлайн — 2026-10-23 (пт).
- Цель пользователя — сдать раньше, если всё сделано корректно. Качество важнее скорости.
