# Статус работы

_Обновлено: 2026-10-16_

## Текущий этап

**Этап 3 — Реализация.** Готово: **BE-01…BE-04** ✅. Следующий тикет — **BE-05** (репозиторий на pgx + sqlc).

| Этап | Статус |
|---|---|
| Бэклоги B1–B3 (`backlog/`) | ✅ одобрены; дизайн получен, копия — `design/design-spec.md` |
| Тикеты BE-01…13, FE-01…10, CM-01 (`tickets/README.md`) | ✅ одобрены вместе с порядком реализации |
| Реализация | ▶️ BE-01 ✅ (PR #1), BE-02 ✅ (PR #2), BE-03 ✅ (PR #3), BE-04 ✅ — ждём коммит/пуш (C6) |

## Следующий шаг

1. Пользователь коммитит и пушит BE-04 (C6 в `commits.md`), сливает в `main`.
2. По сигналу «продолжай» — BE-05 (см. «Уточнения после BE-04» в тикете).

## Локальный запуск инфраструктуры

- `docker compose up -d postgres migrate` — БД + миграции; `docker compose run --rm migrate migrate status|down|reset|up`.
- `docker compose down` — остановить (данные в volume `postgres-data`); `down -v` — удалить и данные.

## Как проверять бэкенд на Windows

Windows Smart App Control блокирует запуск свежесобранных тестовых `.exe` (`An Application Control policy has blocked this file`). Поэтому тесты и линтер гоняем в Docker:
- тесты: `docker run --rm -v "<путь>/backend:/src" -v gomodcache:/go/pkg/mod -w /src golang:1.27 go test -race -count=1 ./...` (Debian-образ: в alpine нет cgo для `-race`; volume `gomodcache` кэширует модули)
- линтер: `docker run --rm -v "<путь>/backend:/src" -w /src golangci/golangci-lint:v2.14.0 golangci-lint run ./...`
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
