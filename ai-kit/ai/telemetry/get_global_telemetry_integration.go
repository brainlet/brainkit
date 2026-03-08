// Ported from: packages/ai/src/telemetry/get-global-telemetry-integration.ts
package telemetry

// BindTelemetryIntegration wraps a telemetry integration so that its methods
// are safe to use as standalone callbacks (preserving any receiver context).
// In Go, this is a no-op since Go closures capture their receiver naturally,
// but we keep it for API parity and to filter nil methods.
func BindTelemetryIntegration(integration TelemetryIntegration) TelemetryIntegration {
	return TelemetryIntegration{
		OnStart:          integration.OnStart,
		OnStepStart:      integration.OnStepStart,
		OnToolCallStart:  integration.OnToolCallStart,
		OnToolCallFinish: integration.OnToolCallFinish,
		OnStepFinish:     integration.OnStepFinish,
		OnFinish:         integration.OnFinish,
	}
}

// GetGlobalTelemetryIntegration creates a factory that merges globally registered
// integrations (via RegisterTelemetryIntegration) with per-call integrations
// into a single composite integration.
//
// Returns a function that accepts local integrations and returns the merged
// TelemetryIntegration.
func GetGlobalTelemetryIntegration() func(integrations []TelemetryIntegration) TelemetryIntegration {
	globalIntegrations := GetGlobalTelemetryIntegrations()

	return func(integrations []TelemetryIntegration) TelemetryIntegration {
		allIntegrations := make([]TelemetryIntegration, 0, len(globalIntegrations)+len(integrations))
		allIntegrations = append(allIntegrations, globalIntegrations...)
		allIntegrations = append(allIntegrations, integrations...)

		createComposite := func(
			getListener func(TelemetryIntegration) Listener,
		) Listener {
			listeners := make([]Listener, 0)
			for _, integration := range allIntegrations {
				l := getListener(integration)
				if l != nil {
					listeners = append(listeners, l)
				}
			}

			return func(event interface{}) error {
				for _, listener := range listeners {
					func() {
						defer func() {
							// Swallow panics from individual integrations
							recover()
						}()
						// Swallow errors from individual integrations
						_ = listener(event)
					}()
				}
				return nil
			}
		}

		return TelemetryIntegration{
			OnStart:          createComposite(func(i TelemetryIntegration) Listener { return i.OnStart }),
			OnStepStart:      createComposite(func(i TelemetryIntegration) Listener { return i.OnStepStart }),
			OnToolCallStart:  createComposite(func(i TelemetryIntegration) Listener { return i.OnToolCallStart }),
			OnToolCallFinish: createComposite(func(i TelemetryIntegration) Listener { return i.OnToolCallFinish }),
			OnStepFinish:     createComposite(func(i TelemetryIntegration) Listener { return i.OnStepFinish }),
			OnFinish:         createComposite(func(i TelemetryIntegration) Listener { return i.OnFinish }),
		}
	}
}
