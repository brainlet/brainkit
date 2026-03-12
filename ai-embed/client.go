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

// StreamTextParams configures a streamText call.
type StreamTextParams struct {
	BaseURL  string
	APIKey   string
	Model    string
	Prompt   string
	OnToken  func(token string) // called for each text delta
}

// StreamTextResult holds the final result of a streamText call.
type StreamTextResult struct {
	Text string `json:"text"`
}

// StreamText calls the AI SDK's streamText function, streaming tokens via OnToken callback.
func (c *Client) StreamText(params StreamTextParams) (*StreamTextResult, error) {
	ctx := c.bridge.Context()

	ctx.SetFunc("__go_stream_token", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("__go_stream_token: missing argument")
		}
		if params.OnToken != nil {
			params.OnToken(args[0].String())
		}
		return this.Context().NewBool(true), nil
	})

	js := fmt.Sprintf(`
		const { streamText, createOpenAI } = globalThis.__ai_sdk;
		const openai = createOpenAI({
			apiKey: %q,
			baseURL: %q,
		});
		const result = streamText({
			model: openai(%q),
			prompt: %q,
		});
		let fullText = "";
		for await (const delta of result.textStream) {
			__go_stream_token(delta);
			fullText += delta;
		}
		JSON.stringify({ text: fullText });
	`, params.APIKey, params.BaseURL, params.Model, params.Prompt)

	val, err := c.bridge.Eval("stream-text.js", qjs.Code(js), qjs.FlagAsync())
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
