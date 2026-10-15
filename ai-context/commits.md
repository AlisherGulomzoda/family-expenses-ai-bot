# Предложенные коммиты

Ассистент предлагает, пользователь коммитит и пушит. Отмечать ✅ после коммита.

| # | Ветка | Сообщение | Что входит | Статус |
|---|---|---|---|---|
| C1 | `main` | `docs: add task context and AI workflow notes` | `.gitignore`, `CLAUDE.md`, `ai-context/` (кроме `private/`) | ✅ `5f905ea` |
| C2 | `main` | `docs: add backlogs, tickets and design spec` | `ai-context/` — бэклоги B2/B3, дизайн, тикеты, статус, решения, правила работы | ✅ `a390259` |
| C3 | `feature/BE-01-skeleton` | `chore(backend): scaffold Go project with config, logging and Dockerfile` | `backend/` (go.mod, cmd/*, internal/config, internal/logger, Dockerfile, Makefile, .golangci.yml, .dockerignore), `.env.example`, `.gitignore`, `ai-context/` (статус BE-01) | ✅ `ff32d9a`, PR #1 |
| C4 | `feature/BE-02-database` | `feat(db): add PostgreSQL schema, goose migrations and compose services` | `backend/migrations/`, `backend/internal/postgres/migrate.go`, `backend/cmd/migrate`, `go.mod/go.sum`, `docker-compose.yml`, `.env.example`, `.gitattributes`, `ai-context/` | ⏳ |
