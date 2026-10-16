# Предложенные коммиты

Ассистент предлагает, пользователь коммитит и пушит. Отмечать ✅ после коммита.

| # | Ветка | Сообщение | Что входит | Статус |
|---|---|---|---|---|
| C1 | `main` | `docs: add task context and AI workflow notes` | `.gitignore`, `CLAUDE.md`, `ai-context/` (кроме `private/`) | ✅ `5f905ea` |
| C2 | `main` | `docs: add backlogs, tickets and design spec` | `ai-context/` — бэклоги B2/B3, дизайн, тикеты, статус, решения, правила работы | ✅ `a390259` |
| C3 | `feature/BE-01-skeleton` | `chore(backend): scaffold Go project with config, logging and Dockerfile` | `backend/` (go.mod, cmd/*, internal/config, internal/logger, Dockerfile, Makefile, .golangci.yml, .dockerignore), `.env.example`, `.gitignore`, `ai-context/` (статус BE-01) | ✅ `ff32d9a`, PR #1 |
| C4 | `feature/BE-02-database` | `feat(db): add PostgreSQL schema, goose migrations and compose services` | `backend/migrations/`, `backend/internal/postgres/migrate.go`, `backend/cmd/migrate`, `go.mod/go.sum`, `docker-compose.yml`, `.env.example`, `.gitattributes`, `ai-context/` | ✅ `49b7105`, PR #2 |
| C5 | `feature/BE-03-openapi` | `docs(api): add OpenAPI contract for expenses REST API` | `backend/api/openapi.yaml`, `backend/api/redocly.yaml`, `backend/Makefile`, `ai-context/` | ✅ `90f281c`, PR #3 |
| C6 | `feature/BE-04-domain-service` | `feat(expense): add domain model, money type and user-scoped service` | `backend/internal/{clock,expense,user,memstore}/`, `backend/Makefile`, `ai-context/` | ✅ `5d657f0`, PR #4 |
| C7 | `feature/BE-05-repository` | `feat(postgres): implement expense and user repositories with contract tests` | `backend/sqlc.yaml`, `backend/internal/postgres/` (queries, sqlcdb, repos, integration test), `backend/internal/expense/expensetest/`, `backend/internal/memstore/`, `go.mod/go.sum`, `Makefile`, `.gitattributes`, `ai-context/` | ✅ `8105d29`, PR #5 |
| C8 | `feature/BE-06-rest-api` | `feat(api): implement REST API for expenses web panel` | `backend/internal/httpapi/`, `backend/cmd/api/`, `go.mod/go.sum`, `docker-compose.yml`, `.env.example`, `ai-context/` | ✅ `1c9b0de`, PR #6 |
| C9 | `feature/BE-07-llm-client` | `feat(llm): add OpenAI-compatible LLM client with retries` | `backend/internal/llm/` (+ `llmtest/`, live test), `backend/Makefile`, `ai-context/` | ✅ `00e4bbe`, PR #7 |
| C10 | `feature/BE-08-agent-tools` | `feat(agent): add tool registry for expense operations` | `backend/internal/agent/tools/`, `go.mod/go.sum`, `ai-context/` | ✅ `801ef5a`, PR #8 |
| C11 | `feature/BE-09-agent-loop` | `feat(agent): implement tool-calling loop with system prompt and chat memory` | `backend/internal/agent/` (agent, prompt, history, agenttest, testdata), `internal/memstore/history.go`, `internal/postgres/` (history, queries, sqlcdb), `internal/expense/service.go`, `ai-context/` | ✅ `2e0325b`, PR #9 |
| C12 | `feature/BE-10-telegram-bot` | `feat(bot): add Telegram bot with long polling and family whitelist` | `backend/internal/telegram/`, `backend/cmd/bot/`, `go.mod/go.sum`, `docker-compose.yml`, `ai-context/` | ✅ `752e1eb`, PR #10 |
| C13 | `feature/BE-12-compose-final` | `chore(compose): finalize docker compose with optional local LLM profile` | `docker-compose.yml`, `.env.example`, `backend/internal/agent/live_test.go`, `backend/Makefile`, `ai-context/` | ✅ `ff188d4`, PR #11 |
| C14 | `feature/FE-01-skeleton` | `chore(frontend): scaffold Vite React app with Tailwind v4, shadcn/ui and design theme` | `frontend/` (без `node_modules`, `dist`), `ai-context/` | ⏳ |
