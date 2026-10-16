// Package llmtest provides a scripted LLM client for tests.
package llmtest

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
)

// Step is one scripted reply: a response or an error.
type Step struct {
	Response llm.Response
	Err      error
}

// Reply is a step with a plain text answer.
func Reply(text string) Step {
	return Step{Response: llm.Response{Message: llm.Message{Role: llm.RoleAssistant, Content: text}, FinishReason: "stop"}}
}

// Call is a step requesting tool calls.
func Call(calls ...llm.ToolCall) Step {
	return Step{Response: llm.Response{Message: llm.Message{Role: llm.RoleAssistant, ToolCalls: calls}, FinishReason: "tool_calls"}}
}

// Fail is a step returning an error.
func Fail(err error) Step { return Step{Err: err} }

// ErrScriptExhausted is returned when the client gets more calls than scripted.
var ErrScriptExhausted = errors.New("llmtest: no more scripted replies")

// Client replies with the scripted steps in order and records every request.
type Client struct {
	mu       sync.Mutex
	steps    []Step
	requests []llm.Request
}

var _ llm.Client = (*Client)(nil)

// New creates a client that replies with steps in order.
func New(steps ...Step) *Client {
	return &Client{steps: steps}
}

// Chat implements llm.Client.
func (c *Client) Chat(_ context.Context, req llm.Request) (llm.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, llm.Request{
		Messages: slices.Clone(req.Messages),
		Tools:    slices.Clone(req.Tools),
	})
	if len(c.steps) == 0 {
		return llm.Response{}, ErrScriptExhausted
	}
	step := c.steps[0]
	c.steps = c.steps[1:]
	return step.Response, step.Err
}

// Requests returns the requests received so far.
func (c *Client) Requests() []llm.Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.requests)
}
