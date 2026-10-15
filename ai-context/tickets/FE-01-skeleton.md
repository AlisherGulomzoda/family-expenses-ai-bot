# FE-01. Каркас фронтенда: Vite, Tailwind v4, shadcn/ui, тема, шрифт

**Оценка:** 2.5 ч · **Зависит от:** — · **Ветка:** `feature/FE-01-skeleton`
**Дизайн:** `design/design-spec.md` §1, §4 (команда `shadcn add`)

## Цель
Пустое приложение с темой из дизайна 1:1, на которое ложатся все экраны.

## Объём
- `frontend/`: Vite + React + TypeScript (`strict`, `noUncheckedIndexedAccess`), алиас `@/` → `src/`.
- Tailwind v4 + `npx shadcn@latest init`, затем **замена темы** блоком `globals.css` из `design-spec.md` §1 целиком (токены `:root` / `.dark`, `chart-1…7` + `-soft` / `-ink`, радиусы, `@theme inline`).
- Установка 28 компонентов одной командой из `design-spec.md` §4.
- Шрифт **Onest** 400/500/600/700: выгрузить `fonts/Onest-*.woff2` из дизайн-системы (https://claude.ai/code/artifact/48d056c2-5924-4259-92c5-5f30842fc16e) в `frontend/public/fonts/`, `@font-face` из спецификации. Лицензия OFL — файл лицензии рядом.
- **next-themes** (`attribute="class"`, `defaultTheme="system"`), провайдер в `main.tsx`.
- Провайдеры: `QueryClientProvider` (TanStack Query, `retry: 1`, `refetchOnWindowFocus: false`), `Toaster` (sonner), `TooltipProvider`.
- `vite.config.ts`: dev-proxy `/api` → `http://localhost:8081` (API опубликован на `127.0.0.1:${API_PORT:-8081}`, см. BE-06; 8080 занят панелью `web`).
- ESLint (typescript-eslint, react-hooks, jsx-a11y) + Prettier (+ `prettier-plugin-tailwindcss`), Vitest + RTL + jsdom.
- `package.json` scripts: `dev`, `build`, `preview`, `lint`, `typecheck`, `test`, `gen:api`.

## Критерии приёмки
- [ ] `npm run build`, `lint`, `typecheck`, `test` проходят.
- [ ] Тестовая страница с Button/Badge/Card показывает цвета, радиусы и шрифт Onest как в дизайне; переключение `.dark` меняет тему.
- [ ] Контраст и фокус-кольцо shadcn не отключены.

## Коммит
`chore(frontend): scaffold Vite React app with Tailwind v4, shadcn/ui and design theme`
