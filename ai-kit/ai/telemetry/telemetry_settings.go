// Ported from: packages/ai/src/telemetry/telemetry-settings.ts
package telemetry

// AttributeValue represents an OpenTelemetry attribute value.
// In Go, this maps to interface{} since it can be string, int, float, bool, or slices thereof.
// TODO: import from an OpenTelemetry Go SDK package if needed
type AttributeValue = interface{}

// Attributes is a map of string keys to attribute values, mirroring OpenTelemetry's Attributes.
// TODO: import from an OpenTelemetry Go SDK package if needed
type Attributes = map[string]AttributeValue

// Tracer represents an OpenTelemetry Tracer interface.
// TODO: import from go.opentelemetry.io/otel/trace
type Tracer interface {
	// StartSpan starts a new span.
	StartSpan(name string, opts ...SpanStartOption) Span
	// StartActiveSpan starts a new span and executes fn within it.
	StartActiveSpan(name string, opts SpanStartOption, fn func(Span) interface{}) interface{}
}

// SpanStartOption configures span creation.
type SpanStartOption struct {
	Attributes Attributes
}

// TelemetrySettings holds telemetry configuration.
type TelemetrySettings struct {
	// IsEnabled enables or disables telemetry. Disabled by default while experimental.
	IsEnabled *bool

	// RecordInputs enables or disables input recording. Enabled by default.
	// You might want to disable input recording to avoid recording sensitive
	// information, to reduce data transfers, or to increase performance.
	RecordInputs *bool

	// RecordOutputs enables or disables output recording. Enabled by default.
	// You might want to disable output recording to avoid recording sensitive
	// information, to reduce data transfers, or to increase performance.
	RecordOutputs *bool

	// FunctionID is an identifier for this function. Used to group telemetry data by function.
	FunctionID *string

	// Metadata is additional information to include in the telemetry data.
	Metadata map[string]AttributeValue

	// TracerInstance is a custom tracer to use for the telemetry data.
	TracerInstance Tracer

	// Integrations are per-call telemetry integrations that receive lifecycle events during generation.
	// These integrations run after any globally registered integrations.
	Integrations []TelemetryIntegration
}
