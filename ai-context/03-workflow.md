# Правила работы с пользователем

## Процесс

1. **Бэклоги** — обсуждаем по очереди: B1 (ТЗ) → B2 (бэкенд: архитектура/технологии) → B3 (фронтенд) → прочие по необходимости.
   Каждый бэклог — файл в `backlog/`, статус: черновик → одобрен.
2. **Тикеты** — после одобрения всех бэклогов декомпозируем на карточки `tickets/BE-xx.md` и `tickets/FE-xx.md`
   (цель, критерии приёмки, зависимости, оценка). Ждём одобрения.
3. **Реализация** — по одобренным тикетам, по одному. После тикета:
   - обновить `00-status.md`;
   - предложить текст коммита (и при необходимости имя ветки) → записать в `commits.md`;
   - пользователь сам коммитит/пушит и даёт знать, что можно продолжать.
4. **Финал** — README, `cover-letter.md` из `future-improvements.md`.

## Git

- Ассистент **не выполняет** `git commit/push/checkout -b`. Только предлагает.
- Формат сообщений: Conventional Commits (`feat(bot): ...`, `fix(api): ...`, `docs: ...`, `chore: ...`), на английском.
- Ветки: `feature/<ticket-id>-<slug>` (например `feature/BE-03-expense-repository`), слияние в `main`.

## Настройка репозитория (один раз)

Локальная папка `tz/` уже содержит файлы → **не клонируем**, а превращаем её в репозиторий:

1. На GitHub: New repository → `family-expenses-ai-bot`, Public, **без** README/.gitignore/license (иначе конфликт при первом пуше).
2. В папке `tz/`:
   ```
   git init -b main
   git add .
   git status        # проверить: нет ai-context/private/ и test-task-expenses.md
   git commit -m "docs: add task context and AI workflow notes"
   git remote add origin https://github.com/<username>/family-expenses-ai-bot.git
   git push -u origin main
   ```

Продолжение с другого устройства: `git clone https://github.com/<username>/family-expenses-ai-bot.git`,
затем вручную скопировать `ai-context/private/` и `test-task-expenses.md` (их нет в репо — не критично для работы).

## Контекст

- После каждого решения/изменения обновлять: `00-status.md`, `02-decisions.md`, при необходимости бэклог/тикет.
- Идеи на будущее — сразу в `future-improvements.md`.
- Язык общения и документов в `ai-context/` — русский. Код, комментарии, коммиты — английский. README — решить в B2/B4.
