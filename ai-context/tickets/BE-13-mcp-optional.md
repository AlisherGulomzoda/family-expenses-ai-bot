# BE-13. (Опционально) MCP-сервер на тех же тулах

**Оценка:** 2 ч · **Зависит от:** BE-08 · **Ветка:** `feature/BE-13-mcp`
**Делаем, только если все основные тикеты (BE + FE) закрыты и время есть.** Иначе — пункт в сопроводительном письме.

## Цель
Показать, что тулы агента протоколо-независимы: тот же реестр доступен любому MCP-клиенту (Claude Desktop и т. п.) — это стек работодателя.

## Объём
- `cmd/mcp` на `github.com/modelcontextprotocol/go-sdk`, транспорт stdio (и/или Streamable HTTP).
- Регистрирует все тулы из `tools.Registry` без дублирования логики.
- Пользователь фиксируется конфигом (`MCP_TELEGRAM_ID`) — MCP-клиент работает от имени одного члена семьи, изоляция сохраняется.
- README: как подключить к Claude Desktop.

## Критерии приёмки
- [ ] MCP Inspector видит 5 тулов с теми же схемами, вызовы работают.
- [ ] Ни строчки бизнес-логики не продублировано.

## Коммит
`feat(mcp): expose expense tools via MCP server`

## Итог реализации (2026-10-16) — ✅ готово

- `internal/mcpserver` — адаптер над `tools.Registry` на официальном `modelcontextprotocol/go-sdk` v1.8.0: каждый тул регистрируется с той же схемой (`json.RawMessage`) и описанием, вызов идёт через `Registry.Call` — та же валидация и тексты ошибок; ошибки — tool result с `isError`, не протокольные. **Бизнес-логики в адаптере нет.**
- `instructions` при подключении: чьи траты, часовой пояс, сегодняшняя дата, как менять/удалять (у бота это в системном промпте). Аннотации: `readOnlyHint` для `list`/`summarize` (новое поле `tools.Tool.ReadOnly`), `openWorldHint: false`.
- `cmd/mcp` — stdio, логи в stderr; `MCP_TELEGRAM_ID` (конфиг валидируется) должен быть зарегистрированным членом семьи, иначе понятная ошибка при старте.
- compose: сервис `mcp` в профиле `mcp` (`docker compose --profile mcp run --rm -T mcp`), `.env.example`, README — раздел с конфигом Claude Desktop и MCP Inspector.
- Проверено: тесты через in-memory транспорт SDK (клиент видит 5 тулов со схемами и описаниями реестра, read-only подсказки; instructions; вызовы пишут траты нужному пользователю; плохие аргументы и чужой id — tool error, чужая трата цела); конфиг-тесты; **живой прогон по stdio в Docker против Postgres** (initialize → tools/list → summarize → delete чужого id); `golangci-lint` — 0; полный набор бэкенда с integration — зелёный.

Отличия от плана:
- Проверка «MCP Inspector видит 5 тулов» заменена тестом с настоящим MCP-клиентом SDK + ручным JSON-RPC по stdio (Inspector требует Node, локально его нет). Команда для Inspector — в README.
- Только stdio; Streamable HTTP — в будущее (нужна авторизация и пользователь из токена).
- Траты через MCP получают `source = bot` (D56).
