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
