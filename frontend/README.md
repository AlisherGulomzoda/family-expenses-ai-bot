# Веб-панель «Семейные расходы»

React 19 + TypeScript (strict) + Vite, Tailwind CSS v4, shadcn/ui (Radix), TanStack Query.
Дизайн и тема — из `ai-context/design/design-spec.md` (Claude Design), шрифт Onest (`public/fonts`, SIL OFL 1.1).

## Команды

| Команда                           | Что делает                                                                         |
| --------------------------------- | ---------------------------------------------------------------------------------- |
| `npm run dev`                     | dev-сервер; `/api` проксируется на `http://localhost:8081` (API из docker compose) |
| `npm run build`                   | проверка типов + production-сборка в `dist/`                                       |
| `npm run preview`                 | раздаёт `dist/`                                                                    |
| `npm run typecheck`               | `tsc -b`                                                                           |
| `npm run lint`                    | oxlint (react, react-hooks, jsx-a11y, typescript, vitest)                          |
| `npm run format` / `format:check` | Prettier (+ сортировка Tailwind-классов)                                           |
| `npm test`                        | Vitest + Testing Library (jsdom)                                                   |
| `npm run gen:api`                 | типы API из `../backend/api/openapi.yaml` → `src/api/schema.d.ts` (коммитится)     |

Без локального Node.js — то же самое в Docker:

```bash
docker run --rm -v "$PWD:/app" -v npmcache:/root/.npm -w /app node:lts-alpine sh -c "npm ci && npm test"
```

Dev-сервер в контейнере: `-p 127.0.0.1:5173:5173`, `-e VITE_API_PROXY_TARGET=http://host.docker.internal:8081`, команда `npx vite --host 0.0.0.0`. На Windows-хосте изменения файлов не доходят до контейнера (bind mount без событий ФС) — после правок `docker restart <контейнер>`, иначе Vite отдаёт устаревшие модули.

**Без бэкенда** — API из MSW-моков (≈40 реалистичных трат четырёх членов семьи, та же семантика фильтров, сортировки и ошибок, что у настоящего API): `VITE_API_MOCKS=true npm run dev`. В production-сборку моки не попадают.

## Production

`Dockerfile`: сборка на `node:24-alpine` → статика на `nginx:1.30-alpine`. nginx (`nginx/default.conf`) отдаёт SPA (любой путь → `index.html`, без кэша; `/assets` — кэш на год), проксирует `/api/` на сервис `api` с `X-Request-ID`, добавляет заголовки безопасности. В compose это сервис `web` на `http://localhost:8080` (`WEB_PORT`; по умолчанию только с этой машины — `WEB_BIND`).

## Структура

- `src/components/ui/` — компоненты shadcn/ui, сгенерированы CLI (`npx shadcn@latest add …`), исключены из линтера и Prettier; точечные правки помечены комментариями (`progress.tsx` — a11y, `dialog.tsx` — «Закрыть»);
- `src/app/` — провайдеры приложения (тема, QueryClient, тултипы, тосты);
- `src/api/` — типизированный клиент (`openapi-fetch`), `ApiError`, хуки TanStack Query (`useExpenses`, `useExpensesInfinite`, `useSummary`, `useUsers`, мутации с инвалидацией);
- `src/mocks/` — MSW: данные (`db.ts`), обработчики (`handlers.ts`), браузер (`browser.ts`) и тесты (`node.ts`);
- `src/shared/` — деньги (`formatRub`, `parseRubInput` — без float), даты (форматы дизайна, быстрые периоды Пн–Вс), категории (`CATEGORIES`, проверены против enum API на этапе компиляции);
- `src/theme/` — переключатель темы;
- `src/index.css` — тема из дизайна: токены shadcn, 7 цветов категорий (`chart-N`, `-soft`, `-ink`), радиусы, шрифт.
