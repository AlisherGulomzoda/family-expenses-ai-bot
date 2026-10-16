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

Без локального Node.js — то же самое в Docker:

```bash
docker run --rm -v "$PWD:/app" -v npmcache:/root/.npm -w /app node:lts-alpine sh -c "npm ci && npm test"
```

Dev-сервер в контейнере: `-p 127.0.0.1:5173:5173`, `-e VITE_API_PROXY_TARGET=http://host.docker.internal:8081`, команда `npx vite --host 0.0.0.0`.

## Структура

- `src/components/ui/` — компоненты shadcn/ui, сгенерированы CLI (`npx shadcn@latest add …`), не редактируются вручную, исключены из линтера и Prettier;
- `src/app/` — провайдеры приложения (тема, QueryClient, тултипы, тосты);
- `src/theme/` — переключатель темы;
- `src/index.css` — тема из дизайна: токены shadcn, 7 цветов категорий (`chart-N`, `-soft`, `-ink`), радиусы, шрифт.
