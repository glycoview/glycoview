package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/shared/constant"
)

// Service drives the tool-using agent loop against an OpenAI-compatible
// chat-completions backend. We point it at Ollama Cloud by default but any
// OpenAI-compatible endpoint works.
type Service struct {
	deps     Deps
	store    SettingsStore
	maxTurns int
}

func NewService(deps Deps, store SettingsStore) *Service {
	return &Service{deps: deps, store: store, maxTurns: 8}
}

// Emit is the sink the HTTP handler provides to push SSE events as they
// happen. Implementations must be cheap and non-blocking; we pass through
// small structured payloads only.
type Emit func(event string, data any)

// systemPrompt keeps the agent's behaviour consistent across turns. It's
// deliberately short: the tools are the interesting surface area.
const systemPrompt = `You are the clinical assistant inside Glycoview, a self-hosted
dashboard for Nightscout-compatible CGM + loop data. The person talking to you
is the patient or their clinician reviewing the patient's own data.

You have read-only access to the real data via tools: glucose readings,
treatments (bolus, SMB, carbs, temp basals, fingersticks), pump-loop status
(Trio / OpenAPS output), therapy profile (CR, ISF, targets, basal),
time-in-range, AGP percentiles and summary statistics. Call the tools — do
not guess. Always call 'now' at the start of a new thread so you know what
'today' means, then call the specific data tool you need with unix-ms
timestamps.

Answer in clean Markdown. When a visual comparison helps, emit a small
self-contained SVG inside a fenced code block labelled 'svg' and the UI will
render it inline — keep SVGs under 600x300 and use currentColor so they
read in both light and dark themes.

Round numbers humanely: mg/dL as integers, percents to one decimal, insulin
to one decimal. Prefer TIR, GMI and eA1C framing; call out hypos and very-
high excursions by duration rather than point-in-time values when possible.
Be concise — clinicians skim; patients want reassurance + the one action
that matters. Never invent data. If a tool returns an empty window, say so.`

// RunChat executes the tool loop until the model is done or maxTurns is hit.
// Content deltas, tool starts and tool ends are pushed through `emit` in
// order. Errors are surfaced as a final "error" event and returned.
func (s *Service) RunChat(ctx context.Context, req ChatRequest, emit Emit) error {
	settings, err := Load(ctx, s.store)
	if err != nil {
		return fmt.Errorf("load ai settings: %w", err)
	}
	if strings.TrimSpace(settings.APIKey) == "" {
		return errors.New("no Ollama API key configured — set one in AI → Settings first")
	}
	model := settings.Model
	if strings.TrimSpace(req.Model) != "" {
		model = req.Model
	}

	client := openai.NewClient(
		option.WithAPIKey(settings.APIKey),
		option.WithBaseURL(settings.BaseURL),
	)

	// Seed the conversation: system prompt + user/assistant history from caller.
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}
	for _, m := range req.Messages {
		converted, err := convertMessage(m)
		if err != nil {
			return err
		}
		messages = append(messages, converted...)
	}

	tools := toolsForAPI()

	for turn := 0; turn < s.maxTurns; turn++ {
		params := openai.ChatCompletionNewParams{
			Model:    model,
			Messages: messages,
			Tools:    tools,
		}

		stream := client.Chat.Completions.NewStreaming(ctx, params)
		acc := openai.ChatCompletionAccumulator{}

		for stream.Next() {
			chunk := stream.Current()
			acc.AddChunk(chunk)
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta
				if delta.Content != "" {
					emit(EventContent, map[string]any{"delta": delta.Content})
				}
			}
		}
		if err := stream.Err(); err != nil {
			return fmt.Errorf("completion stream: %w", err)
		}
		if len(acc.Choices) == 0 {
			return errors.New("empty completion")
		}
		choice := acc.Choices[0]

		// If no tool calls, the assistant is done with this turn.
		if len(choice.Message.ToolCalls) == 0 {
			emit(EventDone, map[string]any{"finish": string(choice.FinishReason)})
			return nil
		}

		// Tool calls: append the assistant turn, run each tool, append the
		// results, loop for another round.
		messages = append(messages, choice.Message.ToParam())

		for _, tc := range choice.Message.ToolCalls {
			callID := tc.ID
			name := tc.Function.Name
			rawArgs := tc.Function.Arguments
			emit(EventToolStart, map[string]any{
				"id":   callID,
				"name": name,
				"args": json.RawMessage(rawArgs),
			})
			started := time.Now()
			result, runErr := s.runTool(ctx, name, []byte(rawArgs))
			duration := time.Since(started)

			var toolMsg openai.ChatCompletionMessageParamUnion
			if runErr != nil {
				msg := fmt.Sprintf(`{"error":%q}`, runErr.Error())
				emit(EventToolEnd, map[string]any{
					"id":         callID,
					"name":       name,
					"error":      runErr.Error(),
					"durationMs": duration.Milliseconds(),
				})
				toolMsg = openai.ToolMessage(msg, callID)
			} else {
				data, _ := json.Marshal(result)
				emit(EventToolEnd, map[string]any{
					"id":         callID,
					"name":       name,
					"result":     json.RawMessage(data),
					"durationMs": duration.Milliseconds(),
				})
				toolMsg = openai.ToolMessage(string(data), callID)
			}
			messages = append(messages, toolMsg)
		}
	}

	emit(EventDone, map[string]any{"finish": "max_turns"})
	return nil
}

// runTool looks up and executes a tool by name.
func (s *Service) runTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	for _, t := range Registry() {
		if t.Name == name {
			return t.Handler(ctx, s.deps, args)
		}
	}
	return nil, fmt.Errorf("unknown tool %q", name)
}

// toolsForAPI converts our internal tool registry into the OpenAI-compatible
// tools parameter expected by /v1/chat/completions.
func toolsForAPI() []openai.ChatCompletionToolUnionParam {
	defs := Registry()
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(defs))
	for _, t := range defs {
		out = append(out, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  t.Schema,
		}))
	}
	return out
}

// convertMessage translates our JSON-friendly ChatMessage into one or more
// OpenAI message param unions. For assistant turns with tool calls we also
// emit the tool result messages caller supplied inline.
func convertMessage(m ChatMessage) ([]openai.ChatCompletionMessageParamUnion, error) {
	switch m.Role {
	case "user":
		return []openai.ChatCompletionMessageParamUnion{openai.UserMessage(m.Content)}, nil
	case "assistant":
		if len(m.ToolCalls) == 0 {
			return []openai.ChatCompletionMessageParamUnion{openai.AssistantMessage(m.Content)}, nil
		}
		calls := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			fn := openai.ChatCompletionMessageFunctionToolCallParam{
				ID:   tc.ID,
				Type: constant.Function("function"),
				Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
					Name:      tc.Name,
					Arguments: string(tc.Args),
				},
			}
			calls = append(calls, openai.ChatCompletionMessageToolCallUnionParam{OfFunction: &fn})
		}
		return []openai.ChatCompletionMessageParamUnion{
			{OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				Content:   openai.ChatCompletionAssistantMessageParamContentUnion{OfString: openai.String(m.Content)},
				ToolCalls: calls,
			}},
		}, nil
	case "tool":
		return []openai.ChatCompletionMessageParamUnion{openai.ToolMessage(m.Content, m.ToolCallID)}, nil
	case "system":
		return []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(m.Content)}, nil
	default:
		return nil, fmt.Errorf("unknown role %q", m.Role)
	}
}
