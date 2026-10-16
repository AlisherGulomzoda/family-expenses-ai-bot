# BE-12. Финальный compose: api, bot, профиль `local-llm`

**Оценка:** 1.5 ч · **Зависит от:** BE-06, BE-10 (сервис `web` добавляется FE-тикетом) · **Ветка:** `feature/BE-12-compose`

## Цель
`docker compose up` на чистой машине поднимает всю систему; локальная LLM — по желанию.

## Объём
- `docker-compose.yml`: `postgres` → `migrate` → `api`, `bot` (+ `web` из FE) с healthchecks и `depends_on` по условиям; `restart: unless-stopped` для долгоживущих сервисов.
- Профиль `local-llm`:
  - `ollama` (`ollama/ollama`, volume для моделей);
  - `ollama-pull` — одноразовый сервис, скачивающий модель (`OLLAMA_MODEL`, по умолчанию Qwen подходящего размера);
  - инструкция: `docker compose --profile local-llm up` + `LLM_BASE_URL=http://ollama:11434/v1`, `LLM_MODEL=...` в `.env`.
- `.env.example` — финальная версия, сгруппирована и прокомментирована (что обязательно: `TELEGRAM_BOT_TOKEN`, `LLM_API_KEY`).
- Проверка «с нуля»: `docker compose down -v` → `cp .env.example .env` → заполнить 2–3 значения → `docker compose up -d` → всё работает.

## Критерии приёмки
- [ ] Чистый запуск по инструкции выше без ручных шагов, кроме заполнения `.env`.
- [ ] `docker compose ps` — все сервисы healthy, `migrate` завершился с 0.
- [ ] Профиль `local-llm` поднимает Ollama, бот отвечает через локальную модель.

## Коммит
`chore(compose): finalize docker compose with optional local LLM profile`

## Итог реализации (2026-10-16) — ✅ готово

Проверено:
- **Чистый запуск** в изолированном проекте (`docker compose -p fe-clean up -d --build`, другие порты, свои тома — данные пользователя не тронуты): от пустых томов до готовности **16 с**; `postgres` healthy → `migrate` exit 0 → `api` healthy, `bot` подключился к Telegram; `/healthz` и `/api/categories` отвечают. Проект удалён вместе с томами.
- **Профиль `local-llm`**: образ `ollama/ollama:0.40.5` (9.37 ГБ — включает GPU-библиотеки), модель `qwen2.5:3b` скачана через `ollama-pull`; тот же LLM-клиент **без изменений кода** (только `LLM_BASE_URL/MODEL/API_KEY`) вызывает тулы через локальную модель.
- `docker compose config` валиден с профилем и без.

Наблюдение для eval и письма — **качество маленькой локальной модели недостаточно**: `qwen2.5:3b` на CPU ~15–30 с на запрос, в минимальном тесте отнесла кофе к `shopping`, на «последнее было не 350, а 450» переспросила вместо исправления. DeepSeek на тех же примерах — 7/7 и ~2 с. Систематическое сравнение (включая `qwen2.5:7b`) — в BE-11.

Что сделано:
- `ollama` + одноразовый `ollama-pull` в профиле `local-llm`; модель хранится в томе `ollama-data`; `bot` ждёт `ollama-pull` через `depends_on ... required: false` (без профиля зависимость игнорируется).
- Версия образа закреплена: `OLLAMA_IMAGE_TAG=0.40.5` (локальный образ помечен этим тегом — повторной загрузки нет).
- `.env.example` переписан в финальном виде: сначала 2 обязательных значения (`TELEGRAM_BOT_TOKEN`, `LLM_API_KEY`), затем семья, LLM (DeepSeek + как переключиться на локальную), агент, Ollama, общее, порты/БД.
- Живой тест примеров ТЗ (`internal/agent/live_test.go`), `make test-live` гоняет и его.

Отличия от плана:
- Тяжёлые загрузки (образ, модель) запускает пользователь сам — правило в `03-workflow.md`.
- Сервис `web` будет добавлен в FE-09.
