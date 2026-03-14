package agentembed

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	quickjs "github.com/buke/quickjs-go"
)

// AgentConfig configures a persistent agent.
type AgentConfig struct {
	Name         string
	Model        string // "provider/model-id" format
	Instructions string
	Tools        map[string]Tool
	MaxSteps     int // default: 5
	Description  string
	ToolChoice   *ToolChoice
}

// GenerateParams configures an Agent.Generate call.
type GenerateParams struct {
	Prompt   string
	Messages []Message
	MaxSteps int // override; 0 = use agent default
}

// StreamParams configures an Agent.Stream call.
type StreamParams struct {
	Prompt   string
	Messages []Message
	MaxSteps int
	OnToken  func(token string)
}

// Agent is a persistent agent handle.
// The underlying Mastra Agent lives in the JS runtime's global registry.
type Agent struct {
	id      string // UUID (with hyphens) — used as JS registry key
	jsID    string // UUID with underscores — used for JS variable names
	name    string
	sandbox *Sandbox
	closed  bool
}

// ID returns the agent's unique identifier.
func (a *Agent) ID() string { return a.id }

// Name returns the agent's display name.
func (a *Agent) Name() string { return a.name }

// Sandbox returns the agent's sandbox.
func (a *Agent) Sandbox() *Sandbox { return a.sandbox }

// CreateAgent creates a persistent agent in this sandbox.
// The agent is stored in globalThis.__agents[id] and reused across calls.
func (s *Sandbox) CreateAgent(cfg AgentConfig) (*Agent, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, fmt.Errorf("agent-embed: sandbox is closed")
	}
	s.mu.Unlock()

	// Generate agent ID
	val, err := s.bridge.Eval("agent-id.js", `crypto.randomUUID()`)
	if err != nil {
		return nil, fmt.Errorf("agent-embed: generate agent ID: %w", err)
	}
	agentID := val.String()
	val.Free()

	// Use a clean ID for JS variable names (replace hyphens)
	jsID := strings.ReplaceAll(agentID, "-", "_")

	// Register Go tool callbacks
	s.registerToolCallbacks(jsID, cfg.Tools)

	// Build tool definitions JS
	toolsJS := "{}"
	if len(cfg.Tools) > 0 {
		var toolDefs []string
		fnName := "__go_tool_" + jsID
		for name, tool := range cfg.Tools {
			schemaJS := "embed.z.object({})"
			if tool.Parameters != nil {
				schemaJS = buildZodSchema(tool.Parameters)
			}

			toolDefs = append(toolDefs, fmt.Sprintf(`%q: embed.createTool({
				id: %q,
				description: %q,
				inputSchema: %s,
				execute: async (input) => {
					const raw = %s(%q, JSON.stringify(input || {}));
					return JSON.parse(raw);
				},
			})`, name, name, tool.Description, schemaJS, fnName, name))
		}
		toolsJS = fmt.Sprintf("{%s}", strings.Join(toolDefs, ",\n"))
	}

	// Resolve model
	modelJS := fmt.Sprintf("%q", cfg.Model)
	provider, modelID := splitModel(cfg.Model)
	if provider != "" {
		if pc, ok := s.providers[provider]; ok {
			opts := map[string]string{"apiKey": pc.APIKey}
			if pc.BaseURL != "" {
				opts["baseURL"] = pc.BaseURL
			}
			optsJSON, _ := json.Marshal(opts)
			factory := providerFactory(provider)
			if factory != "" {
				modelJS = fmt.Sprintf("embed.%s(%s)(%q)", factory, string(optsJSON), modelID)
			}
		}
	}

	maxSteps := cfg.MaxSteps
	if maxSteps == 0 {
		maxSteps = 5
	}

	// Create the Mastra Agent in JS and store in registry
	js := fmt.Sprintf(`(() => {
		const embed = globalThis.__agent_embed;
		const agent = new embed.Agent({
			name: %q,
			id: %q,
			description: %q,
			instructions: %q,
			model: %s,
			tools: %s,
		});
		globalThis.__agents[%q] = agent;
		return "ok";
	})()`,
		cfg.Name, agentID, cfg.Description, cfg.Instructions,
		modelJS, toolsJS, agentID,
	)

	_, err = s.bridge.Eval("create-agent.js", js)
	if err != nil {
		return nil, fmt.Errorf("agent-embed: create agent: %w", err)
	}

	agent := &Agent{
		id:      agentID,
		jsID:    jsID,
		name:    cfg.Name,
		sandbox: s,
	}
	s.registerAgent(agent)
	return agent, nil
}

// Generate calls the agent's generate method.
func (a *Agent) Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	if a.closed {
		return nil, fmt.Errorf("agent-embed: agent %q is closed", a.name)
	}

	maxSteps := params.MaxSteps
	maxStepsJS := ""
	if maxSteps > 0 {
		maxStepsJS = fmt.Sprintf("maxSteps: %d,", maxSteps)
	}

	// Build prompt or messages
	inputJS := ""
	if params.Prompt != "" {
		inputJS = fmt.Sprintf("%q", params.Prompt)
	} else if len(params.Messages) > 0 {
		msgsJSON, _ := json.Marshal(params.Messages)
		inputJS = string(msgsJSON)
	} else {
		return nil, fmt.Errorf("agent-embed: either Prompt or Messages is required")
	}

	js := fmt.Sprintf(`(async () => {
		const agent = globalThis.__agents[%q];
		if (!agent) throw new Error("agent not found: %s");
		const result = await agent.generate(%s, { %s });
		return JSON.stringify({
			text: result.text || "",
			reasoning: result.reasoningText || "",
			toolCalls: (result.toolCalls || []).map(tc => ({
				toolCallId: tc.toolCallId, toolName: tc.toolName, args: tc.args,
			})),
			toolResults: (result.toolResults || []).map(tr => ({
				toolCallId: tr.toolCallId, toolName: tr.toolName, args: tr.args, result: tr.result,
			})),
			finishReason: result.finishReason || "stop",
			usage: {
				promptTokens: result.usage?.inputTokens || result.usage?.promptTokens || 0,
				completionTokens: result.usage?.outputTokens || result.usage?.completionTokens || 0,
				totalTokens: result.usage?.totalTokens || 0,
				reasoningTokens: result.usage?.reasoningTokens || 0,
			},
			steps: (result.steps || []).map(s => ({
				text: s.text || "",
				reasoning: s.reasoningText || "",
				finishReason: s.finishReason || "",
				usage: {
					promptTokens: s.usage?.inputTokens || s.usage?.promptTokens || 0,
					completionTokens: s.usage?.outputTokens || s.usage?.completionTokens || 0,
					totalTokens: s.usage?.totalTokens || 0,
				},
				stepType: s.stepType || "initial",
				isContinued: !!s.isContinued,
				toolCalls: (s.toolCalls || []).map(tc => ({
					toolCallId: tc.toolCallId, toolName: tc.toolName, args: tc.args,
				})),
				toolResults: (s.toolResults || []).map(tr => ({
					toolCallId: tr.toolCallId, toolName: tr.toolName, args: tr.args, result: tr.result,
				})),
			})),
			response: {
				id: result.response?.id || "",
				modelId: result.response?.modelId || "",
				timestamp: result.response?.timestamp?.toISOString?.() || "",
			},
			runId: result.runId || "",
		});
	})()`, a.id, a.id, inputJS, maxStepsJS)

	resultJSON, err := a.sandbox.eval(ctx, "agent-generate.js", js)
	if err != nil {
		return nil, fmt.Errorf("agent-embed: generate: %w", err)
	}

	var result GenerateResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, fmt.Errorf("agent-embed: parse result: %w", err)
	}
	return &result, nil
}

// Stream calls the agent's stream method with real-time token-by-token callbacks.
// Uses Mastra's agent.stream() which returns textStream (AsyncIterable<string>).
// Each text chunk is delivered to OnToken as it arrives via SSE.
func (a *Agent) Stream(ctx context.Context, params StreamParams) (*StreamResult, error) {
	if a.closed {
		return nil, fmt.Errorf("agent-embed: agent %q is closed", a.name)
	}

	// Register streaming callback on globalThis
	callbackName := "__go_stream_" + a.jsID
	qctx := a.sandbox.bridge.Context()
	qctx.Globals().Set(callbackName, qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) > 0 && params.OnToken != nil {
			params.OnToken(args[0].String())
		}
		return qctx.NewBool(true)
	}))

	maxStepsJS := ""
	if params.MaxSteps > 0 {
		maxStepsJS = fmt.Sprintf("maxSteps: %d,", params.MaxSteps)
	}

	inputJS := ""
	if params.Prompt != "" {
		inputJS = fmt.Sprintf("%q", params.Prompt)
	} else if len(params.Messages) > 0 {
		msgsJSON, _ := json.Marshal(params.Messages)
		inputJS = string(msgsJSON)
	} else {
		return nil, fmt.Errorf("agent-embed: either Prompt or Messages is required")
	}

	js := fmt.Sprintf(`(async () => {
		const agent = globalThis.__agents[%q];
		if (!agent) throw new Error("agent not found: %s");
		const stream = await agent.stream(%s, { %s });

		// Deliver text chunks in real-time via Go callback
		for await (const chunk of stream.textStream) {
			%s(chunk);
		}

		// Await final result Promises
		const [text, usage, finishReason] = await Promise.all([
			stream.text,
			stream.usage,
			stream.finishReason,
		]);

		return JSON.stringify({
			text: text || "",
			finishReason: finishReason || "stop",
			usage: {
				promptTokens: usage?.inputTokens || usage?.promptTokens || 0,
				completionTokens: usage?.outputTokens || usage?.completionTokens || 0,
				totalTokens: usage?.totalTokens || 0,
			},
			response: {
				id: stream.traceId || "",
				modelId: "",
				timestamp: "",
			},
			runId: stream.traceId || "",
		});
	})()`, a.id, a.id, inputJS, maxStepsJS, callbackName)

	resultJSON, err := a.sandbox.eval(ctx, "agent-stream.js", js)
	if err != nil {
		return nil, fmt.Errorf("agent-embed: stream: %w", err)
	}

	var result StreamResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, fmt.Errorf("agent-embed: parse stream result: %w", err)
	}
	return &result, nil
}

// Close removes the agent from the JS registry and frees resources.
func (a *Agent) Close() {
	a.close()
	a.sandbox.unregisterAgent(a.id)
}

func (a *Agent) close() {
	if a.closed {
		return
	}
	a.closed = true
	// Remove from JS registry
	a.sandbox.bridge.Eval("close-agent.js", fmt.Sprintf(
		`delete globalThis.__agents[%q]; delete globalThis.__go_tool_%s; delete globalThis.__go_stream_%s`,
		a.id, a.jsID, a.jsID,
	))
}

// buildZodSchema converts a JSON Schema (json.RawMessage) to a Zod expression string.
func buildZodSchema(schema json.RawMessage) string {
	var s map[string]any
	if err := json.Unmarshal(schema, &s); err != nil {
		return "embed.z.object({})"
	}
	return buildZodObject(s)
}

func buildZodObject(schema map[string]any) string {
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		return "embed.z.object({})"
	}

	required := map[string]bool{}
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				required[s] = true
			}
		}
	}

	var fields []string
	for name, propRaw := range props {
		prop, ok := propRaw.(map[string]any)
		if !ok {
			continue
		}

		typ, _ := prop["type"].(string)
		desc, _ := prop["description"].(string)

		var zodType string
		switch typ {
		case "number", "integer":
			zodType = "embed.z.number()"
		case "boolean":
			zodType = "embed.z.boolean()"
		case "array":
			items, _ := prop["items"].(map[string]any)
			if items != nil {
				zodType = fmt.Sprintf("embed.z.array(%s)", buildZodObject(items))
			} else {
				zodType = "embed.z.array(embed.z.any())"
			}
		case "object":
			zodType = buildZodObject(prop)
		default:
			zodType = "embed.z.string()"
		}

		if desc != "" {
			zodType += fmt.Sprintf(".describe(%q)", desc)
		}
		if !required[name] {
			zodType += ".optional()"
		}

		fields = append(fields, fmt.Sprintf("%q: %s", name, zodType))
	}

	return fmt.Sprintf("embed.z.object({%s})", strings.Join(fields, ", "))
}

// splitModel parses "provider/model-id" format.
func splitModel(model string) (provider, modelID string) {
	idx := strings.IndexByte(model, '/')
	if idx < 0 {
		return "", model
	}
	return model[:idx], model[idx+1:]
}

// providerFactory returns the JS factory function name for a provider.
func providerFactory(provider string) string {
	factories := map[string]string{
		"openai":     "createOpenAI",
		"anthropic":  "createAnthropic",
		"google":     "createGoogleGenerativeAI",
		"mistral":    "createMistral",
		"xai":        "createXai",
		"groq":       "createGroq",
		"deepseek":   "createDeepSeek",
		"cerebras":   "createCerebras",
		"perplexity": "createPerplexity",
		"togetherai": "createTogetherAI",
		"fireworks":  "createFireworks",
		"cohere":     "createCohere",
	}
	return factories[provider]
}
