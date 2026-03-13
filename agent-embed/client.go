package agentembed

import (
	"encoding/json"
	"fmt"
	"net/http"

	quickjs "github.com/buke/quickjs-go"

	"github.com/brainlet/brainkit/jsbridge"
)

// ClientConfig configures an agent-embed client.
type ClientConfig struct {
	HTTPClient *http.Client // optional; defaults to http.DefaultClient
}

// Client wraps a jsbridge.Bridge with a loaded Mastra bundle.
type Client struct {
	bridge *jsbridge.Bridge
}

// NewClient creates a Client with all polyfills and the Mastra bundle loaded.
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
		return nil, fmt.Errorf("agent-embed: create bridge: %w", err)
	}

	if err := LoadBundle(b); err != nil {
		b.Close()
		return nil, err
	}

	return &Client{bridge: b}, nil
}

// Close shuts down the client and frees all resources.
func (c *Client) Close() {
	if c.bridge != nil {
		c.bridge.Close()
	}
}

// Bridge returns the underlying jsbridge.Bridge.
func (c *Client) Bridge() *jsbridge.Bridge { return c.bridge }

// ToolDef defines a tool that the agent can call.
// The Execute function is called from JS via a Go bridge function.
type ToolDef struct {
	ID          string
	Description string
	InputSchema map[string]interface{} // JSON Schema object for tool parameters
	Execute     func(args map[string]interface{}) (interface{}, error)
}

// GenerateParams configures an agent generate call.
type GenerateParams struct {
	Provider     string // e.g. "openai"
	APIKey       string // required
	BaseURL      string // optional; overrides default provider URL
	Model        string // required; e.g. "openai/gpt-4o-mini"
	Instructions string
	Prompt       string
	Tools        []ToolDef
}

// GenerateResult holds the result of an agent generate call.
type GenerateResult struct {
	Text string `json:"text"`
}

// Generate creates a Mastra agent with the given config and generates a response.
func (c *Client) Generate(params GenerateParams) (*GenerateResult, error) {
	if params.APIKey == "" {
		return nil, fmt.Errorf("agent-embed: APIKey is required")
	}
	if params.Model == "" {
		return nil, fmt.Errorf("agent-embed: Model is required")
	}

	ctx := c.bridge.Context()

	// Register Go tool functions as __go_tool_<id>
	toolRegistrations := ""
	for _, tool := range params.Tools {
		tool := tool // capture
		fnName := "__go_tool_" + tool.ID
		ctx.Globals().Set(fnName, ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			var input map[string]interface{}
			if len(args) > 0 {
				json.Unmarshal([]byte(args[0].String()), &input)
			}
			result, err := tool.Execute(input)
			if err != nil {
				return qctx.NewString(`{"error":"` + err.Error() + `"}`)
			}
			b, _ := json.Marshal(result)
			return qctx.NewString(string(b))
		}))

		// Build Zod schema from Go InputSchema
		schemaJS := "embed.z.object({})"
		if tool.InputSchema != nil {
			schemaJS = buildZodSchema(tool.InputSchema)
		}

		toolRegistrations += fmt.Sprintf(`
		tools[%q] = embed.createTool({
			id: %q,
			description: %q,
			inputSchema: %s,
			execute: async (input) => {
				const raw = %s(JSON.stringify(input || {}));
				return JSON.parse(raw);
			},
		});
		`, tool.ID, tool.ID, tool.Description, schemaJS, fnName)
	}

	// Set up provider env vars so Mastra auto-resolves the model string
	envSetup := ""
	if params.Provider != "" {
		envKey := providerEnvKey(params.Provider)
		if envKey != "" {
			envSetup += fmt.Sprintf("globalThis.process.env.%s = %q;\n", envKey, params.APIKey)
		}
	}

	// If a custom BaseURL is provided, use createOpenAI explicitly
	modelExpr := fmt.Sprintf("%q", params.Model)
	if params.BaseURL != "" {
		modelExpr = fmt.Sprintf(`(() => {
			const openai = embed.createOpenAI({
				apiKey: %q,
				baseURL: %q,
			});
			return openai.chat(%q);
		})()`, params.APIKey, params.BaseURL, extractModelName(params.Model))
	}

	js := fmt.Sprintf(`(async () => {
		const embed = globalThis.__agent_embed;
		%s
		const tools = {};
		%s

		const agent = new embed.Agent({
			name: "go-agent",
			instructions: %q,
			model: %s,
			tools: tools,
		});

		const result = await agent.generate(%q);
		return JSON.stringify({ text: result.text });
	})()`,
		envSetup,
		toolRegistrations,
		params.Instructions,
		modelExpr,
		params.Prompt,
	)

	val, err := c.bridge.Eval("agent-generate.js", js, quickjs.EvalAwait(true))
	if err != nil {
		return nil, fmt.Errorf("agent-embed: generate: %w", err)
	}
	defer val.Free()

	var result GenerateResult
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		return nil, fmt.Errorf("agent-embed: parse result %q: %w", val.String(), err)
	}
	return &result, nil
}

// buildZodSchema converts a JSON Schema object to a Zod schema expression string.
func buildZodSchema(schema map[string]interface{}) string {
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return "embed.z.object({})"
	}

	result := "embed.z.object({"
	first := true
	for name, propRaw := range props {
		prop, ok := propRaw.(map[string]interface{})
		if !ok {
			continue
		}
		if !first {
			result += ", "
		}
		first = false

		typ, _ := prop["type"].(string)
		desc, _ := prop["description"].(string)
		zodType := "embed.z.string()"
		switch typ {
		case "number", "integer":
			zodType = "embed.z.number()"
		case "boolean":
			zodType = "embed.z.boolean()"
		case "array":
			zodType = "embed.z.array(embed.z.any())"
		case "object":
			zodType = "embed.z.object({})"
		}
		if desc != "" {
			zodType += fmt.Sprintf(".describe(%q)", desc)
		}
		result += fmt.Sprintf("%q: %s", name, zodType)
	}
	result += "})"
	return result
}

// providerEnvKey returns the environment variable name for a provider's API key.
func providerEnvKey(provider string) string {
	switch provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "google":
		return "GOOGLE_GENERATIVE_AI_API_KEY"
	case "mistral":
		return "MISTRAL_API_KEY"
	case "groq":
		return "GROQ_API_KEY"
	case "xai":
		return "XAI_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	case "cerebras":
		return "CEREBRAS_API_KEY"
	case "perplexity":
		return "PERPLEXITY_API_KEY"
	case "together":
		return "TOGETHER_AI_API_KEY"
	case "fireworks":
		return "FIREWORKS_API_KEY"
	case "cohere":
		return "COHERE_API_KEY"
	}
	return ""
}

// extractModelName gets the model name from "provider/model" format.
func extractModelName(model string) string {
	for i := 0; i < len(model); i++ {
		if model[i] == '/' {
			return model[i+1:]
		}
	}
	return model
}
