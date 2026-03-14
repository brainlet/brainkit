package aiembed

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	quickjs "github.com/buke/quickjs-go"

	"github.com/brainlet/brainkit/jsbridge"
)

// ClientConfig configures an AI SDK client.
type ClientConfig struct {
	HTTPClient      *http.Client    // optional; defaults to http.DefaultClient
	DefaultProvider *ProviderConfig // optional; default provider for all calls
	EnvVars         map[string]string // optional; env vars for API key resolution
}

// Client wraps a jsbridge.Bridge with a loaded AI SDK bundle.
type Client struct {
	bridge          *jsbridge.Bridge
	defaultProvider *ProviderConfig
	envVars         map[string]string
}

// NewClient creates a Client with all polyfills and the AI SDK bundle loaded.
func NewClient(cfg ClientConfig) (*Client, error) {
	fetchOpts := []jsbridge.FetchOption{}
	if cfg.HTTPClient != nil {
		fetchOpts = append(fetchOpts, jsbridge.FetchClient(cfg.HTTPClient))
	}

	b, err := jsbridge.New(jsbridge.Config{},
		jsbridge.Console(),
		jsbridge.Encoding(),
		jsbridge.Streams(),
		jsbridge.Crypto(),
		jsbridge.URL(),
		jsbridge.Timers(),
		jsbridge.Abort(),
		jsbridge.Events(),
		jsbridge.StructuredClone(),
		jsbridge.Fetch(fetchOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("ai-embed: create bridge: %w", err)
	}

	if err := LoadBundle(b); err != nil {
		b.Close()
		return nil, err
	}

	return &Client{
		bridge:          b,
		defaultProvider: cfg.DefaultProvider,
		envVars:         cfg.EnvVars,
	}, nil
}

// Close shuts down the client and frees all resources.
func (c *Client) Close() {
	if c.bridge != nil {
		c.bridge.Close()
	}
}

// Bridge returns the underlying jsbridge.Bridge for advanced use.
func (c *Client) Bridge() *jsbridge.Bridge { return c.bridge }

// GenerateText calls the AI SDK's generateText function.
func (c *Client) GenerateText(params GenerateTextParams) (*GenerateTextResult, error) {
	modelJS, err := buildProviderJS(params.Model, c.defaultProvider, c.envVars)
	if err != nil {
		return nil, fmt.Errorf("ai-embed: generateText: %w", err)
	}

	// Register Go tool callbacks if tools are provided
	toolsJS := ""
	if len(params.Tools) > 0 {
		// Register the Go callback for tool execution
		ctx := c.bridge.Context()
		ctx.Globals().Set("__go_execute_tool", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.ThrowError(fmt.Errorf("__go_execute_tool: expected 2 args (toolName, argsJSON)"))
			}
			toolName := args[0].String()
			argsJSON := args[1].String()

			tool, ok := params.Tools[toolName]
			if !ok || tool.Execute == nil {
				return qctx.ThrowError(fmt.Errorf("tool %q not found or has no Execute function", toolName))
			}

			result, err := tool.Execute(json.RawMessage(argsJSON))
			if err != nil {
				return qctx.ThrowError(fmt.Errorf("tool %q execution error: %w", toolName, err))
			}

			resultJSON, err := json.Marshal(result)
			if err != nil {
				return qctx.ThrowError(fmt.Errorf("tool %q result marshal error: %w", toolName, err))
			}
			return qctx.NewString(string(resultJSON))
		}))

		// Build JS tool definitions
		var toolDefs []string
		for name, tool := range params.Tools {
			hasExecute := tool.Execute != nil
			executeJS := "undefined"
			if hasExecute {
				executeJS = fmt.Sprintf(`async (args) => {
					const resultJSON = __go_execute_tool(%q, JSON.stringify(args));
					return JSON.parse(resultJSON);
				}`, name)
			}
			toolDefs = append(toolDefs, fmt.Sprintf(`%q: __ai_sdk.tool({
				description: %q,
				parameters: __ai_sdk.jsonSchema(%s),
				execute: %s,
			})`, name, tool.Description, string(tool.Parameters), executeJS))
		}
		toolsJS = fmt.Sprintf("opts.tools = {%s};", strings.Join(toolDefs, ",\n"))
	}

	// Build the prompt/messages payload
	paramsJSON, err := json.Marshal(struct {
		Prompt          string                            `json:"prompt,omitempty"`
		System          string                            `json:"system,omitempty"`
		Messages        []Message                         `json:"messages,omitempty"`
		MaxSteps        int                               `json:"maxSteps,omitempty"`
		ProviderOptions map[string]map[string]interface{} `json:"providerOptions,omitempty"`
	}{
		Prompt:          params.Prompt,
		System:          params.System,
		Messages:        params.Messages,
		MaxSteps:        params.MaxSteps,
		ProviderOptions: params.ProviderOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("ai-embed: marshal params: %w", err)
	}

	callSettings := buildCallSettingsJS(params.CallSettings)

	// Build toolChoice JS if specified
	toolChoiceJS := ""
	if params.ToolChoice != nil {
		switch params.ToolChoice.Mode {
		case "auto", "none", "required":
			toolChoiceJS = fmt.Sprintf(`opts.toolChoice = %q;`, params.ToolChoice.Mode)
		case "tool":
			toolChoiceJS = fmt.Sprintf(`opts.toolChoice = {type: "tool", toolName: %q};`, params.ToolChoice.ToolName)
		}
	}

	js := fmt.Sprintf(`(async () => {
		const params = %s;
		const opts = {
			model: %s,
			%s
		};
		if (params.prompt) opts.prompt = params.prompt;
		if (params.system) opts.system = params.system;
		if (params.messages) opts.messages = params.messages;
		if (params.maxSteps) opts.maxSteps = params.maxSteps;
		if (params.providerOptions) opts.providerOptions = params.providerOptions;
		%s
		%s

		const result = await __ai_sdk.generateText(opts);
		return JSON.stringify({
			text: result.text || "",
			reasoning: result.reasoning || "",
			finishReason: result.finishReason || "stop",
			usage: result.usage || {},
			response: {
				id: result.response?.id || "",
				modelId: result.response?.modelId || "",
				timestamp: result.response?.timestamp?.toISOString?.() || "",
			},
			toolCalls: (result.toolCalls || []).map(tc => ({
				toolCallId: tc.toolCallId,
				toolName: tc.toolName,
				args: tc.args,
			})),
			toolResults: (result.toolResults || []).map(tr => ({
				toolCallId: tr.toolCallId,
				toolName: tr.toolName,
				args: tr.args,
				result: tr.result,
			})),
			steps: (result.steps || []).map(s => ({
				text: s.text || "",
				finishReason: s.finishReason || "",
				usage: s.usage || {},
				stepType: s.stepType || "",
				isContinued: !!s.isContinued,
				toolCalls: (s.toolCalls || []).map(tc => ({
					toolCallId: tc.toolCallId,
					toolName: tc.toolName,
					args: tc.args,
				})),
				toolResults: (s.toolResults || []).map(tr => ({
					toolCallId: tr.toolCallId,
					toolName: tr.toolName,
					args: tr.args,
					result: tr.result,
				})),
			})),
		});
	})()`, string(paramsJSON), modelJS, callSettings, toolsJS, toolChoiceJS)

	val, err := c.bridge.Eval("generate-text.js", js, quickjs.EvalAwait(true))
	if err != nil {
		return nil, fmt.Errorf("ai-embed: generateText: %w", err)
	}
	defer val.Free()

	var result GenerateTextResult
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		return nil, fmt.Errorf("ai-embed: parse result: %w", err)
	}
	return &result, nil
}

// StreamText calls the AI SDK's streamText function, streaming tokens via callbacks.
func (c *Client) StreamText(params StreamTextParams) (*StreamTextResult, error) {
	modelJS, err := buildProviderJS(params.Model, c.defaultProvider, c.envVars)
	if err != nil {
		return nil, fmt.Errorf("ai-embed: streamText: %w", err)
	}

	ctx := c.bridge.Context()

	// Register token callback
	ctx.Globals().Set("__go_stream_token", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) > 0 && params.OnToken != nil {
			params.OnToken(args[0].String())
		}
		return qctx.NewBool(true)
	}))

	// Build the prompt/messages payload
	paramsJSON, err := json.Marshal(struct {
		Prompt          string                            `json:"prompt,omitempty"`
		System          string                            `json:"system,omitempty"`
		Messages        []Message                         `json:"messages,omitempty"`
		MaxSteps        int                               `json:"maxSteps,omitempty"`
		ProviderOptions map[string]map[string]interface{} `json:"providerOptions,omitempty"`
	}{
		Prompt:          params.Prompt,
		System:          params.System,
		Messages:        params.Messages,
		MaxSteps:        params.MaxSteps,
		ProviderOptions: params.ProviderOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("ai-embed: marshal params: %w", err)
	}

	callSettings := buildCallSettingsJS(params.CallSettings)

	js := fmt.Sprintf(`(async () => {
		const params = %s;
		const opts = {
			model: %s,
			%s
		};
		if (params.prompt) opts.prompt = params.prompt;
		if (params.system) opts.system = params.system;
		if (params.messages) opts.messages = params.messages;
		if (params.maxSteps) opts.maxSteps = params.maxSteps;
		if (params.providerOptions) opts.providerOptions = params.providerOptions;

		const result = __ai_sdk.streamText(opts);
		let fullText = "";
		for await (const part of result.fullStream) {
			if (part.type === "text-delta") {
				__go_stream_token(part.textDelta);
				fullText += part.textDelta;
			}
		}
		const usage = await result.usage;
		const finishReason = await result.finishReason;
		const response = await result.response;
		return JSON.stringify({
			text: fullText,
			finishReason: finishReason || "stop",
			usage: {
				promptTokens: usage?.promptTokens || 0,
				completionTokens: usage?.completionTokens || 0,
				totalTokens: usage?.totalTokens || 0,
			},
			response: {
				id: response?.id || "",
				modelId: response?.modelId || "",
				timestamp: response?.timestamp?.toISOString?.() || "",
			},
		});
	})()`, string(paramsJSON), modelJS, callSettings)

	val, err := c.bridge.Eval("stream-text.js", js, quickjs.EvalAwait(true))
	if err != nil {
		return nil, fmt.Errorf("ai-embed: streamText: %w", err)
	}
	defer val.Free()

	var result StreamTextResult
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		return nil, fmt.Errorf("ai-embed: parse streamText result: %w", err)
	}
	return &result, nil
}

// GenerateObject calls the AI SDK's generateObject function.
func (c *Client) GenerateObject(params GenerateObjectParams) (*GenerateObjectResult, error) {
	modelJS, err := buildProviderJS(params.Model, c.defaultProvider, c.envVars)
	if err != nil {
		return nil, fmt.Errorf("ai-embed: generateObject: %w", err)
	}

	schemaJSON := string(params.Schema)
	if schemaJSON == "" {
		return nil, fmt.Errorf("ai-embed: generateObject: schema is required")
	}

	// Build params for the JS side
	paramsJSON, err := json.Marshal(struct {
		Prompt            string                            `json:"prompt,omitempty"`
		System            string                            `json:"system,omitempty"`
		Messages          []Message                         `json:"messages,omitempty"`
		SchemaName        string                            `json:"schemaName,omitempty"`
		SchemaDescription string                            `json:"schemaDescription,omitempty"`
		Mode              string                            `json:"mode,omitempty"`
		ProviderOptions   map[string]map[string]interface{} `json:"providerOptions,omitempty"`
	}{
		Prompt:            params.Prompt,
		System:            params.System,
		Messages:          params.Messages,
		SchemaName:        params.SchemaName,
		SchemaDescription: params.SchemaDescription,
		Mode:              params.Mode,
		ProviderOptions:   params.ProviderOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("ai-embed: marshal params: %w", err)
	}

	callSettings := buildCallSettingsJS(params.CallSettings)

	js := fmt.Sprintf(`(async () => {
		const params = %s;
		const schema = __ai_sdk.jsonSchema(%s);
		const opts = {
			model: %s,
			schema: schema,
			%s
		};
		if (params.prompt) opts.prompt = params.prompt;
		if (params.system) opts.system = params.system;
		if (params.messages) opts.messages = params.messages;
		if (params.schemaName) opts.schemaName = params.schemaName;
		if (params.schemaDescription) opts.schemaDescription = params.schemaDescription;
		if (params.mode) opts.mode = params.mode;
		if (params.providerOptions) opts.providerOptions = params.providerOptions;

		const result = await __ai_sdk.generateObject(opts);
		return JSON.stringify({
			object: result.object,
			finishReason: result.finishReason || "stop",
			usage: result.usage || {},
			response: {
				id: result.response?.id || "",
				modelId: result.response?.modelId || "",
				timestamp: result.response?.timestamp?.toISOString?.() || "",
			},
		});
	})()`, string(paramsJSON), schemaJSON, modelJS, callSettings)

	val, err := c.bridge.Eval("generate-object.js", js, quickjs.EvalAwait(true))
	if err != nil {
		return nil, fmt.Errorf("ai-embed: generateObject: %w", err)
	}
	defer val.Free()

	var result GenerateObjectResult
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		return nil, fmt.Errorf("ai-embed: parse generateObject result: %w", err)
	}
	return &result, nil
}

// Embed embeds a single value using an embedding model.
func (c *Client) Embed(params EmbedParams) (*EmbedResult, error) {
	modelJS, err := buildEmbeddingProviderJS(params.Model, c.defaultProvider, c.envVars)
	if err != nil {
		return nil, fmt.Errorf("ai-embed: embed: %w", err)
	}

	valueJSON, _ := json.Marshal(params.Value)

	js := fmt.Sprintf(`(async () => {
		const result = await __ai_sdk.embed({
			model: %s,
			value: %s,
		});
		return JSON.stringify({
			embedding: result.embedding,
			usage: { tokens: result.usage?.tokens || 0 },
		});
	})()`, modelJS, string(valueJSON))

	val, err := c.bridge.Eval("embed.js", js, quickjs.EvalAwait(true))
	if err != nil {
		return nil, fmt.Errorf("ai-embed: embed: %w", err)
	}
	defer val.Free()

	var result EmbedResult
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		return nil, fmt.Errorf("ai-embed: parse embed result: %w", err)
	}
	return &result, nil
}

// EmbedMany embeds multiple values using an embedding model.
func (c *Client) EmbedMany(params EmbedManyParams) (*EmbedManyResult, error) {
	modelJS, err := buildEmbeddingProviderJS(params.Model, c.defaultProvider, c.envVars)
	if err != nil {
		return nil, fmt.Errorf("ai-embed: embedMany: %w", err)
	}

	valuesJSON, _ := json.Marshal(params.Values)

	js := fmt.Sprintf(`(async () => {
		const result = await __ai_sdk.embedMany({
			model: %s,
			values: %s,
		});
		return JSON.stringify({
			embeddings: result.embeddings,
			usage: { tokens: result.usage?.tokens || 0 },
		});
	})()`, modelJS, string(valuesJSON))

	val, err := c.bridge.Eval("embed-many.js", js, quickjs.EvalAwait(true))
	if err != nil {
		return nil, fmt.Errorf("ai-embed: embedMany: %w", err)
	}
	defer val.Free()

	var result EmbedManyResult
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		return nil, fmt.Errorf("ai-embed: parse embedMany result: %w", err)
	}
	return &result, nil
}
