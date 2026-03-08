// Ported from: packages/ai/src/telemetry/get-tracer.ts
package telemetry

// GetTracerOptions configures which tracer to use.
type GetTracerOptions struct {
	// IsEnabled controls whether telemetry is enabled.
	IsEnabled bool
	// TracerInstance is a custom tracer to use.
	TracerInstance Tracer
}

// GetTracer returns a Tracer based on the provided options.
// If telemetry is not enabled, returns NoopTracer.
// If a custom tracer is provided, returns that.
// Otherwise, returns a default tracer (NoopTracer in this implementation).
func GetTracer(opts *GetTracerOptions) Tracer {
	if opts == nil || !opts.IsEnabled {
		return NoopTracer
	}

	if opts.TracerInstance != nil {
		return opts.TracerInstance
	}

	// In the TypeScript version, this calls trace.getTracer('ai') from @opentelemetry/api.
	// In Go, the caller should provide a tracer via opts.TracerInstance.
	// Default to NoopTracer.
	return NoopTracer
}
