// Package tools are the functions the LLM agent can call to work with the
// expenses of the person it talks to.
//
// Every handler receives an expense.UserScope: the owner is fixed by the
// Telegram update before the model is involved, and no tool schema has a
// user field. Whatever the model writes, it can only touch that user's data.
//
// Tools do not depend on a protocol: the agent calls them directly and the
// same registry could be served over MCP.
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
)

// Handler executes a tool. args is the raw JSON produced by the model.
type Handler func(ctx context.Context, scope expense.UserScope, args []byte) (any, error)

// Tool is a function available to the model.
type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage // JSON Schema of the arguments object
	Handler     Handler
}

// Registry is a set of tools.
type Registry struct {
	tools []Tool
}

// NewRegistry creates a registry of the given tools.
func NewRegistry(tools ...Tool) *Registry {
	return &Registry{tools: tools}
}

// Default returns the registry of all expense tools.
func Default() *Registry {
	return NewRegistry(addExpense(), listExpenses(), summarizeExpenses(), updateExpense(), deleteExpense())
}

// Tools returns the tools in registration order.
func (r *Registry) Tools() []Tool {
	return slices.Clone(r.tools)
}

// Defs describes the tools for the LLM.
func (r *Registry) Defs() []llm.ToolDef {
	defs := make([]llm.ToolDef, len(r.tools))
	for i, t := range r.tools {
		defs[i] = llm.ToolDef{Name: t.Name, Description: t.Description, Parameters: t.Schema}
	}
	return defs
}

// Result is what a tool call produced.
type Result struct {
	// Content is the JSON sent back to the model: the tool's output, or
	// {"error": ...} that the model can react to (fix arguments, ask the user).
	Content string
	// Err is set when the call failed. Internal reports a failure that is
	// not the model's fault (e.g. the database is down).
	Err      error
	Internal bool
}

// Call runs the named tool. It never panics on bad model output: every
// problem becomes an error result for the model.
func (r *Registry) Call(ctx context.Context, scope expense.UserScope, name, args string) Result {
	i := slices.IndexFunc(r.tools, func(t Tool) bool { return t.Name == name })
	if i < 0 {
		err := fmt.Errorf("unknown tool %q", name)
		return errorResult(err, "Неизвестный инструмент "+name, nil, false)
	}
	if args == "" {
		args = "{}"
	}
	out, err := r.tools[i].Handler(ctx, scope, []byte(args))
	if err != nil {
		return classify(err)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return errorResult(err, "Внутренняя ошибка. Попробуй ещё раз позже", nil, true)
	}
	return Result{Content: string(raw)}
}

// argError is a problem with the arguments the model produced.
type argError struct {
	msg    string
	fields map[string]string
}

func (e *argError) Error() string { return e.msg }

func badArgs(msg string) error { return &argError{msg: msg} }

func classify(err error) Result {
	var ae *argError
	var ve *expense.ValidationError
	switch {
	case errors.As(err, &ae):
		return errorResult(err, ae.msg, ae.fields, false)
	case errors.As(err, &ve):
		return errorResult(err, "Некорректные данные траты", ve.Fields, false)
	case errors.Is(err, expense.ErrNotFound):
		return errorResult(err, "Трата не найдена у этого пользователя", nil, false)
	default:
		return errorResult(err, "Внутренняя ошибка. Попробуй ещё раз позже", nil, true)
	}
}

func errorResult(err error, msg string, fields map[string]string, internal bool) Result {
	body := struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields,omitempty"`
	}{msg, fields}
	raw, _ := json.Marshal(body) // strings only, cannot fail
	return Result{Content: string(raw), Err: err, Internal: internal}
}
