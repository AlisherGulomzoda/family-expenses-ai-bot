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
