package plugin

import (
	"context"
	"encoding/json"
	"github.com/brainlet/brainkit/internal/syncx"

	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/sdk"
)

// Plugin is the builder returned by New(). Authors register capabilities on it.
// The manifest is assembled automatically from registrations.
type Plugin struct {
	owner       string
	name        string
	version     string
	description string

	mu            syncx.Mutex
	tools         []toolRegistration
	subscriptions []subscriptionRegistration
	events        []eventRegistration
	interceptors  []interceptorRegistration

	onStartFn func(Client) error
	onStopFn  func() error
}

// PluginOption configures optional Plugin settings.
type PluginOption func(*Plugin)

// WithDescription sets the plugin description.
func WithDescription(desc string) PluginOption {
	return func(p *Plugin) {
		p.description = desc
	}
}

// New creates a new plugin builder.
func New(owner, name, version string, opts ...PluginOption) *Plugin {
	p := &Plugin{
		owner:   owner,
		name:    name,
		version: version,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// OnStart registers a callback invoked after handshake + manifest processing.
func (p *Plugin) OnStart(fn func(Client) error) {
	p.onStartFn = fn
}

// OnStop registers a callback invoked before shutdown.
func (p *Plugin) OnStop(fn func() error) {
	p.onStopFn = fn
}

// --- Internal registration types ---

type toolRegistration struct {
	name        string
	description string
	inputSchema string
	handler     func(ctx context.Context, client Client, input json.RawMessage) (json.RawMessage, error)
}

type subscriptionRegistration struct {
	topic   string
	handler func(ctx context.Context, payload json.RawMessage, client Client)
}

type eventRegistration struct {
	name        string
	description string
	schema      string
}

type interceptorRegistration struct {
	name        string
	priority    int
	topicFilter string
	handler     func(ctx context.Context, msg InterceptMessage) (*InterceptMessage, error)
}

// --- Generic registration functions ---

// Tool registers a typed tool handler. Schema is auto-generated from In.
func Tool[In, Out any](p *Plugin, name, description string, handler func(ctx context.Context, client Client, in In) (Out, error)) {
	var zero In
	schema := string(tools.StructToJSONSchema(zero))

	p.mu.Lock()
	defer p.mu.Unlock()

	p.tools = append(p.tools, toolRegistration{
		name:        name,
		description: description,
		inputSchema: schema,
		handler: func(ctx context.Context, client Client, input json.RawMessage) (json.RawMessage, error) {
			var in In
			if err := json.Unmarshal(input, &in); err != nil {
				return nil, err
			}
			out, err := handler(ctx, client, in)
			if err != nil {
				return nil, err
			}
			return json.Marshal(out)
		},
	})
}

// On registers a typed event subscription handler.
func On[E any](p *Plugin, topic string, handler func(ctx context.Context, event E, client Client)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.subscriptions = append(p.subscriptions, subscriptionRegistration{
		topic: topic,
		handler: func(ctx context.Context, payload json.RawMessage, client Client) {
			var event E
			if err := json.Unmarshal(payload, &event); err != nil {
				return
			}
			handler(ctx, event, client)
		},
	})
}

// Event declares an event type this plugin will publish.
func Event[E sdk.BrainkitMessage](p *Plugin, description string) {
	var zero E
	schema := string(tools.StructToJSONSchema(zero))
	topic := zero.BusTopic()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.events = append(p.events, eventRegistration{
		name:        topic,
		description: description,
		schema:      schema,
	})
}

// Intercept registers a message interceptor at the given priority.
func Intercept(p *Plugin, name string, priority int, topicFilter string, handler func(ctx context.Context, msg InterceptMessage) (*InterceptMessage, error)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.interceptors = append(p.interceptors, interceptorRegistration{
		name:        name,
		priority:    priority,
		topicFilter: topicFilter,
		handler:     handler,
	})
}

// buildManifest assembles a PluginManifest from all registrations.
func (p *Plugin) buildManifest() PluginManifest {
	p.mu.Lock()
	defer p.mu.Unlock()

	m := PluginManifest{
		Owner:       p.owner,
		Name:        p.name,
		Version:     p.version,
		Description: p.description,
	}

	for _, t := range p.tools {
		m.Tools = append(m.Tools, ToolDefinition{
			Name:        t.name,
			Description: t.description,
			InputSchema: t.inputSchema,
		})
	}

	for _, s := range p.subscriptions {
		m.Subscriptions = append(m.Subscriptions, SubscriptionDefinition{
			Topic: s.topic,
		})
	}

	for _, e := range p.events {
		m.Events = append(m.Events, EventDefinition{
			Name:        e.name,
			Description: e.description,
			Schema:      e.schema,
		})
	}

	for _, i := range p.interceptors {
		m.Interceptors = append(m.Interceptors, InterceptorDefinition{
			Name:        i.name,
			Priority:    i.priority,
			TopicFilter: i.topicFilter,
		})
	}

	return m
}
