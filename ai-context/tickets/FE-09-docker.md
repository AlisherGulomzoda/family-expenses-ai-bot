# FE-09. Docker-образ фронтенда и сервис `web`

**Оценка:** 1 ч · **Зависит от:** FE-01, BE-06 · **Ветка:** `feature/FE-09-docker`

## Цель
Фронтенд собирается и раздаётся в compose; `web` — единственная точка входа для браузера.

## Объём
- `frontend/Dockerfile`: `node:lts-alpine` (`npm ci`, `npm run build`) → `nginx:alpine` с `dist/`.
- `frontend/nginx.conf`: SPA-fallback на `index.html`; `location /api/` → `proxy_pass http://api:8080`; кэш статики с хэшами (`/assets/` — `immutable`), `index.html` — без кэша; gzip.
- Сервис `web` в `docker-compose.yml`: порт `${WEB_PORT:-8080}:80` (добавить `WEB_PORT` в `.env.example`), `depends_on: api (service_healthy)`; nginx проксирует `/api/` на `http://api:8080` и передаёт `X-Request-ID` (`$request_id`) — API подхватывает его в логи.
- `.dockerignore` (`node_modules`, `dist`).

## Критерии приёмки
- [ ] `docker compose up` → http://localhost:8080 открывает панель с реальными данными API.
- [ ] Прямой заход по URL с query-параметрами (фильтры) работает (SPA-fallback).
- [ ] Образ без исходников и `node_modules`.

## Коммит
`chore(frontend): add nginx Docker image and web service to compose`

## Итог реализации (2026-10-16) — ✅ готово

Проверено в compose (`docker compose up -d --build web`): `web` healthy; `/` и любые пути/ссылки с фильтрами → `index.html` (200, `Cache-Control: no-cache`); `/assets/*` — `public, max-age=31536000, immutable` + gzip; шрифты — `font/woff2`, кэш 30 дней; `X-Content-Type-Options`/`X-Frame-Options`/`Referrer-Policy` на всех ответах; `/api/*` проксируется, `X-Request-ID` от nginx виден в логах API; неизвестный `/api/...` → JSON 404 от API; сквозной POST → Postgres (`source: web`) → DELETE 204; в образе только `dist` (2 ассета, шрифты), без исходников и `node_modules`; MSW-воркер вырезан. Образ — 103 МБ.

Что сделано:
- `frontend/Dockerfile` — `node:24-alpine` (`npm ci` с кэшем, `npm run build` = typecheck + сборка, удаление `mockServiceWorker.js`) → `nginx:1.30-alpine`; `HEALTHCHECK` через `wget /nginx-health`.
- `frontend/nginx/default.conf` — SPA-fallback (`try_files $uri /index.html`), кэш ассетов/шрифтов, gzip, `server_tokens off`, `client_max_body_size 64k`; `/api/` → `http://api:8080` через переменную + `resolver 127.0.0.11` (переразрешение имени — переживает перезапуск `api`), `X-Request-ID $request_id`, `X-Forwarded-*`.
- `frontend/nginx/security-headers.conf` — подключается в каждом `location` (nginx не наследует `add_header` сервера в location со своими заголовками).
- Compose: сервис `web`, `depends_on: api (service_healthy)`, порт `${WEB_BIND:-127.0.0.1}:${WEB_PORT:-8080}:80`.
- `.env.example`: `WEB_PORT`, `WEB_BIND`.

Решения / отличия от плана:
- **Панель по умолчанию доступна только с этой машины** (`WEB_BIND=127.0.0.1`, D48): по ТЗ у неё нет авторизации; `WEB_BIND=0.0.0.0` открывает её в локальной сети (например, для телефона). Отметить в README и письме.
- Версии образов закреплены (`node:24-alpine`, `nginx:1.30-alpine`).
- Healthcheck `web` — встроенный в образ (`HEALTHCHECK`), а не в compose.
