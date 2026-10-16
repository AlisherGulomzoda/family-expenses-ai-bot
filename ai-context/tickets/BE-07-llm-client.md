# BE-07. LLM-клиент (OpenAI-совместимый, DeepSeek)

**Оценка:** 2 ч · **Зависит от:** BE-01 · **Ветка:** `feature/BE-07-llm-client`

## Цель
Тонкая абстракция над LLM: агент не знает, какой провайдер за ней. Смена модели — только через `.env`.

## Объём
- `internal/llm`:
  - типы: `Message{Role, Content, ToolCalls, ToolCallID}`, `ToolCall{ID, Name, Arguments json.RawMessage}`, `ToolDef{Name, Description, Parameters json.RawMessage}`, `Response{Message, Usage{PromptTokens, CompletionTokens}, FinishReason}`;
  - интерфейс `Client { Chat(ctx, ChatRequest) (Response, error) }`;
  - реализация `openai.go` на `github.com/openai/openai-go` с `option.WithBaseURL` / `WithAPIKey`: модель, температура, тулы, `tool_choice: auto`;
  - таймаут из конфига, ретраи на `429`/`5xx`/сетевые ошибки (до 2 повторов, экспоненциальная задержка), без ретраев на `4xx`;
  - лог каждого вызова: модель, длительность, токены, кол-во tool calls (без содержимого сообщений на уровне `info`, с содержимым — на `debug`).
- `llmtest.FakeClient` — скриптуемый fake для тестов агента (очередь заранее заданных ответов + запись входящих запросов).

## Критерии приёмки
- [ ] Тест на `httptest`-сервере, имитирующем OpenAI Chat Completions: обычный ответ, ответ с несколькими tool calls, `429` → ретрай → успех, `400` → ошибка без ретрая, таймаут.
- [ ] Ручная проверка с реальным ключом DeepSeek (`deepseek-chat`): простой tool call проходит (скрипт/тест под тегом `live`).
- [ ] Переключение на Ollama (`LLM_BASE_URL=http://ollama:11434/v1`) не требует изменений кода.

## Коммит
`feat(llm): add OpenAI-compatible LLM client with retries`

## Итог реализации (2026-10-16) — ✅ готово, ✅ живая проверка (после пополнения баланса)

Проверено: `go test -race` — llm покрыт на 92.4% (httptest-сервер, имитирующий Chat Completions: текст, несколько tool calls, обратная отправка tool calls/результатов, 429→200, 5xx до исчерпания, 400/401/402/422 без ретраев, таймаут попытки → ретрай, отменённый контекст → без ретраев, пустой/битый ответ, стабильный JSON `Message`); `golangci-lint --build-tags integration,live` — 0.

**Живая проверка с DeepSeek** (`make test-live` / тест под тегом `live`): ✅ пройдена после пополнения баланса: «кофе 350» → `add_expense{"amount":"350","category":"food","comment":"кофе"}` (362/71 токенов), после результата тула — «Записал: кофе — 350 ₽ (Еда).»; ~2 с на два вызова. До пополнения API отвечал `402 Insufficient Balance` — клиент корректно не ретраил и пробросил сообщение.

Отличия от плана:
- **Свой компактный клиент вместо SDK `openai-go`** (D35): нужен один эндпоинт; SDK тянет большие сгенерированные типы и свою политику ретраев. Свой клиент (~250 строк) — явные ретраи/логи, совместимость с DeepSeek/Ollama по одному `base_url`, всё покрыто тестами.
- `ToolCall.Arguments` — строка, а не `json.RawMessage`: модель может вернуть невалидный JSON, его разбирает и отвергает тул (BE-08), а не транспорт/хранилище.
- Ретраи: `MaxRetries` (поставим 2 в BE-10), экспоненциальная пауза с джиттером (0.5s, 1s, … ≤10s), учитывается `Retry-After` (≤30s). Транспортные ошибки — отдельный тип `transportError`.
- `llmtest.Client` — скриптуемый фейк (`Reply`, `Call`, `Fail`) с записью запросов — для тестов агента.
- Логи: на `info` — модель, длительность, попытки, токены, число tool calls (без текста); на `debug` — содержимое ответа.
