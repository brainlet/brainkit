package aiembed

import (
	"encoding/json"
	"fmt"
	"net/http"

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
			})),
		});
	})()`, string(paramsJSON), modelJS, callSettings)

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

// StreamText calls the AI SDK's streamText function, streaming tokens via OnToken callback.
// TODO(task5): Rewrite with jsgen helpers, messages, model routing, rich results.
func (c *Client) StreamText(params StreamTextParams) (*StreamTextResult, error) {
	provider, modelID, apiKey, baseURL := resolveModel(params.Model, c.defaultProvider, c.envVars)
	_ = provider

	if apiKey == "" {
		return nil, fmt.Errorf("ai-embed: streamText: no API key")
	}

	ctx := c.bridge.Context()
	ctx.Globals().Set("__go_stream_token", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) > 0 && params.OnToken != nil {
			params.OnToken(args[0].String())
		}
		return qctx.NewBool(true)
	}))

	// Build provider opts — only include baseURL if set
	streamProviderOpts := fmt.Sprintf(`{apiKey: %q}`, apiKey)
	if baseURL != "" {
		streamProviderOpts = fmt.Sprintf(`{apiKey: %q, baseURL: %q}`, apiKey, baseURL)
	}

	js := fmt.Sprintf(`(async () => {
		const { streamText, createOpenAI } = globalThis.__ai_sdk;
		const openai = createOpenAI(%s);
		const result = streamText({
			model: openai(%q),
			prompt: %q,
		});
		let fullText = "";
		for await (const delta of result.textStream) {
			__go_stream_token(delta);
			fullText += delta;
		}
		return JSON.stringify({ text: fullText });
	})()`, streamProviderOpts, modelID, params.Prompt)

	val, err := c.bridge.Eval("stream-text.js", js, quickjs.EvalAwait(true))
	if err != nil {
		return nil, fmt.Errorf("ai-embed: streamText: %w", err)
	}
	defer val.Free()

	var result StreamTextResult
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		return nil, fmt.Errorf("ai-embed: parse streamText result %q: %w", val.String(), err)
	}
	return &result, nil
}

// GenerateObject calls the AI SDK's generateObject function.
func (c *Client) GenerateObject(params GenerateObjectParams) (*GenerateObjectResult, error) {
	return nil, fmt.Errorf("ai-embed: GenerateObject not yet implemented")
}

// Embed embeds a single value using an embedding model.
func (c *Client) Embed(params EmbedParams) (*EmbedResult, error) {
	return nil, fmt.Errorf("ai-embed: Embed not yet implemented")
}

// EmbedMany embeds multiple values using an embedding model.
func (c *Client) EmbedMany(params EmbedManyParams) (*EmbedManyResult, error) {
	return nil, fmt.Errorf("ai-embed: EmbedMany not yet implemented")
}
