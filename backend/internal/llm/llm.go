// Package llm is a thin, provider-independent interface to a chat model
// with tool calling. The agent depends on Client only; which provider is
// behind it (DeepSeek, a local Ollama, ...) is a matter of configuration.
package llm

import (
	"context"
	"encoding/json"
)

// Role of a chat message.
type Role string

// Message roles.
const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is one chat message. The JSON form is used to store the
// conversation history, so it must stay backward compatible.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content,omitempty"`
	// ToolCalls are requested by the assistant.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// ToolCallID links a tool result to the call it answers.
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolCall is a function call requested by the model.
type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Arguments is the JSON object produced by the model. It is kept as a
	// string because the model may produce invalid JSON; tools validate it.
	Arguments string `json:"arguments"`
}

// ToolDef describes a tool the model may call.
type ToolDef struct {
	Name        string
	Description string
	// Parameters is a JSON Schema of the arguments object.
	Parameters json.RawMessage
}

// Request is one chat completion request. The model and sampling settings
// come from the client configuration.
type Request struct {
	Messages []Message
	Tools    []ToolDef
}

// Usage is the token usage of one completion.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

// Response is the assistant's reply.
type Response struct {
	Message      Message
	FinishReason string
	Usage        Usage
}

// Client sends chat completion requests.
type Client interface {
	Chat(ctx context.Context, req Request) (Response, error)
}
