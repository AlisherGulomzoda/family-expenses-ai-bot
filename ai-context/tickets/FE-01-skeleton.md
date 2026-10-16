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

## Итог реализации (2026-10-16) — ✅ готово, ⏳ визуальная сверка пользователем

Проверено (всё в `node:lts-alpine`, Node 24 — локального Node нет): `format:check` ✓, `typecheck` ✓, `lint` — 0, `test` — 2/2 (рендер + переключение темы классом `.dark` на `<html>`), `build` ✓ (JS 332 КБ / 105 КБ gzip, CSS 86 КБ / 15 КБ gzip). `vite preview` отдаёт страницу и шрифт `font/woff2`. Визуальная сверка с дизайном — пользователь (встроенного браузера в сессии нет; Playwright-образ >1 ГБ — по правилу тяжёлых загрузок не качаем).

Что сделано:
- Vite 8 + React 19 + TypeScript 6 (`strict`, `noUncheckedIndexedAccess`, `noImplicitOverride`), алиас `@/`.
- Tailwind v4 (`@tailwindcss/vite`), shadcn/ui **на Radix** (`radix-nova`), 28 компонентов из спецификации (+ зависимости `toggle`, `separator`).
- Тема из `design-spec.md` §1 1:1 (токены `:root`/`.dark`, `chart-1…7` + `-soft`/`-ink`, радиусы); `shadcn/tailwind.css` сохранён (служебные варианты новых компонентов).
- Шрифт Onest 400–700 скачан из дизайн-системы (проверены сигнатуры `wOF2`) в `public/fonts` + `OFL.txt`.
- Провайдеры: next-themes (`class`, system), TanStack Query (`retry: 1`, без refetch on focus, `staleTime 30s`), Tooltip, Sonner.
- Временная страница сверки темы (шапка, переключатель темы, карточка «Итого», 7 бейджей категорий) — заменится в FE-03.
- Vitest + Testing Library + jsdom, Prettier (+ `prettier-plugin-tailwindcss`), oxlint; dev-прокси `/api` → `http://localhost:8081` (`VITE_API_PROXY_TARGET` для контейнера).

Отличия от плана:
- **oxlint вместо ESLint** — дефолт нынешнего шаблона Vite; плагины react / react-hooks / jsx-a11y / typescript / vitest встроены.
- **shadcn по умолчанию ставит Base UI** (`base-nova`) — переинициализирован на Radix (`-b radix`), т. к. спецификация дизайна написана под Radix (`data-[state=on]`, `AlertDialogCancel`).
- shadcn теперь использует пакет `cn` (shadcn-ui/cn) вместо `clsx` + `tailwind-merge`.
- `src/components/ui/**` (сгенерировано CLI) исключено из oxlint и Prettier: там есть a11y-замечания линтера к вендорному коду — доступность реальных экранов проверим в FE-10 через axe.
- Шаблонные `App.css`, `assets/`, Geist и Vite-favicon удалены; favicon — своя иконка в цвете `primary`.
- Тесты идут ~37 с: больше половины — запуск jsdom на смонтированной из Windows папке; на CI/Linux быстрее.
