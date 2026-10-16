// Package mcpserver serves the agent's expense tools over the Model Context
// Protocol: any MCP client (Claude Desktop, MCP Inspector, another agent)
// works with the expenses of one family member exactly like the Telegram
// agent does — the same registry, schemas, validation and error results.
package mcpserver

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent/tools"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
)

// Version is reported to MCP clients.
const Version = "1.0.0"

// New creates an MCP server with every tool of reg. All calls act on behalf
// of scope's user: as in the bot, no tool can address another family member.
func New(reg *tools.Registry, scope expense.UserScope, userName string, log *slog.Logger) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "family-expenses", Title: "Семейные расходы", Version: Version},
		&mcp.ServerOptions{Instructions: instructions(scope, userName)},
	)
	for _, t := range reg.Tools() {
		closed := false
		server.AddTool(&mcp.Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Schema,
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: t.ReadOnly, OpenWorldHint: &closed},
		}, handler(reg, scope, t.Name, log))
	}
	return server
}

// handler runs one tool through the registry. Problems with the arguments or
// a foreign expense come back as a tool error the model can react to, as in
// the agent loop; they are not protocol errors.
func handler(reg *tools.Registry, scope expense.UserScope, name string, log *slog.Logger) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res := reg.Call(ctx, scope, name, string(req.Params.Arguments))
		switch {
		case res.Internal:
			log.ErrorContext(ctx, "tool call failed", "tool", name, "error", res.Err)
		case res.Err != nil:
			log.InfoContext(ctx, "tool call rejected", "tool", name, "error", res.Err)
		default:
			log.InfoContext(ctx, "tool call", "tool", name)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: res.Content}},
			IsError: res.Err != nil,
		}, nil
	}
}

// instructions replace what the bot's system prompt tells the model: whose
// expenses these are and how dates work. MCP clients get them on connect.
func instructions(scope expense.UserScope, userName string) string {
	return fmt.Sprintf(`Учёт семейных расходов. Инструменты работают только с тратами пользователя %s; траты других членов семьи недоступны.
Суммы — в рублях строкой ("350", "890.50"), итоги считает summarize_expenses — не складывай суммы сам.
Даты — ГГГГ-ММ-ДД в часовом поясе %s; на момент подключения сегодня %s. Если дату не передать, add_expense запишет трату на сегодня.
Чтобы изменить или удалить трату, найди её id через list_expenses и вызови update_expense или delete_expense.`,
		userName, scope.Location(), scope.Today().Format("2006-01-02"))
}
