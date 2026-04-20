package ai

import (
	"encoding/json"
	"time"
)

// ChatMessage mirrors the OpenAI-compatible chat message shape our agent
// loop rebuilds between turns. `content` is the assistant/user/tool text;
// tool_calls/tool_call_id carry the round-trip for function calls.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Args     json.RawMessage `json:"arguments"`
	Result   json.RawMessage `json:"result,omitempty"`
	Error    string          `json:"error,omitempty"`
	Duration time.Duration   `json:"-"`
}

type ChatRequest struct {
	Messages     []ChatMessage `json:"messages"`
	Model        string        `json:"model,omitempty"`        // optional override
	UserTimeZone string        `json:"userTimeZone,omitempty"` // IANA name, e.g. Europe/Berlin
}

type Settings struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"` // returned obfuscated except when admin is editing
	Model   string `json:"model"`
}

// DefaultSettings applied when nothing is configured yet.
func DefaultSettings() Settings {
	return Settings{
		BaseURL: "https://ollama.com/v1",
		Model:   "gpt-oss:120b",
	}
}

// Public contract for the stream events the frontend consumes.
type StreamEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Event type names.
const (
	EventContent   = "content"    // assistant text delta
	EventToolStart = "tool_start" // tool call beginning
	EventToolEnd   = "tool_end"   // tool call result
	EventDone      = "done"       // assistant finished (turn or overall)
	EventError     = "error"      // fatal error
)
