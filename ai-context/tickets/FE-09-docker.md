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
