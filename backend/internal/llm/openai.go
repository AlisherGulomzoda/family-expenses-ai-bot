package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// OpenAIConfig configures a client of the OpenAI-compatible Chat Completions
// API (POST {BaseURL}/chat/completions). DeepSeek, Ollama, vLLM, OpenRouter
// and OpenAI itself all speak it.
type OpenAIConfig struct {
	BaseURL     string // e.g. https://api.deepseek.com or http://ollama:11434/v1
	APIKey      string
	Model       string
	Temperature float64
	// Timeout bounds one attempt; retries get their own timeout.
	Timeout time.Duration
	// MaxRetries is the number of retries after the first attempt on
	// rate limits, server errors and network failures.
	MaxRetries int
	// HTTPClient defaults to a plain http.Client.
	HTTPClient *http.Client
	Log        *slog.Logger
}

// OpenAI is a Client for OpenAI-compatible APIs.
type OpenAI struct {
	cfg      OpenAIConfig
	endpoint string
	// backoff returns the pause before retry n (1-based); replaced in tests.
	backoff func(n int) time.Duration
}

var _ Client = (*OpenAI)(nil)

// NewOpenAI creates a client.
func NewOpenAI(cfg OpenAIConfig) *OpenAI {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{}
	}
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.DiscardHandler)
	}
	return &OpenAI{
		cfg:      cfg,
		endpoint: strings.TrimSuffix(cfg.BaseURL, "/") + "/chat/completions",
		backoff:  exponentialBackoff,
	}
}

// APIError is an error response of the provider.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("llm api: status %d: %s", e.StatusCode, e.Message)
}

func (e *APIError) retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= http.StatusInternalServerError
}

// transportError is a failure to get any HTTP response: network errors
// and per-attempt timeouts. Such calls are retried.
type transportError struct {
	err error
}

func (e *transportError) Error() string { return "llm call: " + e.err.Error() }

func (e *transportError) Unwrap() error { return e.err }

// Chat implements Client.
func (c *OpenAI) Chat(ctx context.Context, req Request) (Response, error) {
	body, err := json.Marshal(c.wireRequest(req))
	if err != nil {
		return Response{}, fmt.Errorf("encode request: %w", err)
	}

	start := time.Now()
	for attempt := 0; ; attempt++ {
		resp, retryAfter, err := c.send(ctx, body)
		if err == nil {
			c.cfg.Log.InfoContext(ctx, "llm call",
				"model", c.cfg.Model,
				"duration_ms", time.Since(start).Milliseconds(),
				"attempts", attempt+1,
				"prompt_tokens", resp.Usage.PromptTokens,
				"completion_tokens", resp.Usage.CompletionTokens,
				"tool_calls", len(resp.Message.ToolCalls),
				"finish_reason", resp.FinishReason,
			)
			c.cfg.Log.DebugContext(ctx, "llm reply", "content", resp.Message.Content, "tool_calls", resp.Message.ToolCalls)
			return resp, nil
		}
		if attempt >= c.cfg.MaxRetries || !isRetryable(ctx, err) {
			return Response{}, err
		}
		pause := c.backoff(attempt + 1)
		if retryAfter > pause {
			pause = retryAfter
		}
		c.cfg.Log.WarnContext(ctx, "llm call failed, retrying", "error", err, "attempt", attempt+1, "pause", pause.String())
		select {
		case <-ctx.Done():
			return Response{}, fmt.Errorf("llm call: %w", ctx.Err())
		case <-time.After(pause):
		}
	}
}

// send makes one attempt. retryAfter is the server's Retry-After hint.
func (c *OpenAI) send(ctx context.Context, body []byte) (resp Response, retryAfter time.Duration, err error) {
	if c.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.cfg.Timeout)
		defer cancel()
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return Response{}, 0, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	httpResp, err := c.cfg.HTTPClient.Do(httpReq)
	if err != nil {
		return Response{}, 0, &transportError{err: err}
	}
	defer httpResp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, 10<<20))
	if err != nil {
		return Response{}, 0, fmt.Errorf("read llm response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		apiErr := &APIError{StatusCode: httpResp.StatusCode, Message: errorMessage(raw)}
		return Response{}, parseRetryAfter(httpResp.Header.Get("Retry-After")), apiErr
	}

	var wire wireResponse
	if err := json.Unmarshal(raw, &wire); err != nil {
		return Response{}, 0, fmt.Errorf("decode llm response: %w", err)
	}
	if len(wire.Choices) == 0 {
		return Response{}, 0, errors.New("llm response has no choices")
	}
	return wire.toResponse(), 0, nil
}

// isRetryable: rate limits, server errors and network failures, but not
// the caller giving up (its context is done).
func isRetryable(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.retryable()
	}
	// Network errors and per-attempt timeouts.
	var tErr *transportError
	return errors.As(err, &tErr)
}

func exponentialBackoff(n int) time.Duration {
	base := 500 * time.Millisecond << (n - 1)             // 0.5s, 1s, 2s, ...
	jitter := time.Duration(rand.Int64N(int64(base / 2))) //nolint:gosec // jitter, not security
	return min(base+jitter, 10*time.Second)
}

func parseRetryAfter(v string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || secs <= 0 {
		return 0
	}
	return min(time.Duration(secs)*time.Second, 30*time.Second)
}

// errorMessage extracts {"error":{"message":...}} or falls back to the raw body.
func errorMessage(raw []byte) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &body) == nil && body.Error.Message != "" {
		return body.Error.Message
	}
	s := strings.TrimSpace(string(raw))
	if len(s) > 500 {
		s = s[:500] + "..."
	}
	return s
}

// Wire format of the Chat Completions API.

type wireFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type wireToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function wireFunctionCall `json:"function"`
}

type wireMessage struct {
	Role       Role           `json:"role"`
	Content    *string        `json:"content"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type wireFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type wireTool struct {
	Type     string          `json:"type"`
	Function wireFunctionDef `json:"function"`
}

type wireRequest struct {
	Model       string        `json:"model"`
	Messages    []wireMessage `json:"messages"`
	Tools       []wireTool    `json:"tools,omitempty"`
	ToolChoice  string        `json:"tool_choice,omitempty"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type wireResponse struct {
	Choices []struct {
		Message      wireMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (c *OpenAI) wireRequest(req Request) wireRequest {
	out := wireRequest{
		Model:       c.cfg.Model,
		Messages:    make([]wireMessage, len(req.Messages)),
		Temperature: c.cfg.Temperature,
	}
	for i, m := range req.Messages {
		content := m.Content
		wm := wireMessage{Role: m.Role, Content: &content, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			wm.ToolCalls = append(wm.ToolCalls, wireToolCall{
				ID: tc.ID, Type: "function", Function: wireFunctionCall{Name: tc.Name, Arguments: tc.Arguments},
			})
		}
		out.Messages[i] = wm
	}
	for _, t := range req.Tools {
		out.Tools = append(out.Tools, wireTool{
			Type: "function", Function: wireFunctionDef(t),
		})
	}
	if len(out.Tools) > 0 {
		out.ToolChoice = "auto"
	}
	return out
}

func (w wireResponse) toResponse() Response {
	choice := w.Choices[0]
	msg := Message{Role: RoleAssistant}
	if choice.Message.Content != nil {
		msg.Content = *choice.Message.Content
	}
	for _, tc := range choice.Message.ToolCalls {
		msg.ToolCalls = append(msg.ToolCalls, ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: tc.Function.Arguments})
	}
	return Response{
		Message:      msg,
		FinishReason: choice.FinishReason,
		Usage:        Usage{PromptTokens: w.Usage.PromptTokens, CompletionTokens: w.Usage.CompletionTokens},
	}
}
