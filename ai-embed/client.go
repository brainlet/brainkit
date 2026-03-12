package aiembed

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/brainlet/brainkit/jsbridge"
	"github.com/fastschema/qjs"
)

// ClientConfig configures an AI SDK client.
type ClientConfig struct {
	HTTPClient *http.Client // optional; defaults to http.DefaultClient
}

// Client wraps a jsbridge.Bridge with a loaded AI SDK bundle.
type Client struct {
	bridge *jsbridge.Bridge
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
		jsbridge.Crypto(),
		jsbridge.URL(),
		jsbridge.Timers(),
		jsbridge.Abort(),
		jsbridge.Events(),
		jsbridge.Fetch(fetchOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("ai-embed: create bridge: %w", err)
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

// Bridge returns the underlying jsbridge.Bridge for advanced use.
func (c *Client) Bridge() *jsbridge.Bridge { return c.bridge }

// GenerateTextParams configures a generateText call.
type GenerateTextParams struct {
	BaseURL string // e.g., "https://api.openai.com/v1" or mock server URL
	APIKey  string
	Model   string // e.g., "gpt-4"
	Prompt  string
}

// GenerateTextResult holds the result of a generateText call.
type GenerateTextResult struct {
	Text string `json:"text"`
}

// GenerateText calls the AI SDK's generateText function.
func (c *Client) GenerateText(params GenerateTextParams) (*GenerateTextResult, error) {
	js := fmt.Sprintf(`
		const { generateText, createOpenAI } = globalThis.__ai_sdk;
		const openai = createOpenAI({
			apiKey: %q,
			baseURL: %q,
		});
		const result = await generateText({
			model: openai(%q),
			prompt: %q,
		});
		JSON.stringify({ text: result.text });
	`, params.APIKey, params.BaseURL, params.Model, params.Prompt)

	val, err := c.bridge.Eval("generate-text.js", qjs.Code(js), qjs.FlagAsync())
	if err != nil {
		return nil, fmt.Errorf("ai-embed: generateText: %w", err)
	}
	defer val.Free()

	var result GenerateTextResult
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		return nil, fmt.Errorf("ai-embed: parse result %q: %w", val.String(), err)
	}
	return &result, nil
}
