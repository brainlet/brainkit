// Ported from: packages/ai/src/telemetry/telemetry-integration-registry.test.ts
package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterTelemetryIntegration(t *testing.T) {
	t.Run("adds an integration to the global registry", func(t *testing.T) {
		ResetGlobalTelemetryIntegrations()

		called := false
		integration := TelemetryIntegration{
			OnStart: func(event interface{}) error { called = true; return nil },
		}

		RegisterTelemetryIntegration(integration)

		result := GetGlobalTelemetryIntegrations()
		assert.Equal(t, 1, len(result))
		// Verify the function is callable
		_ = result[0].OnStart(nil)
		assert.True(t, called)
	})

	t.Run("adds multiple integrations in registration order", func(t *testing.T) {
		ResetGlobalTelemetryIntegrations()

		calls := make([]string, 0)
		integration1 := TelemetryIntegration{
			OnStart: func(event interface{}) error { calls = append(calls, "first"); return nil },
		}
		integration2 := TelemetryIntegration{
			OnFinish: func(event interface{}) error { calls = append(calls, "second"); return nil },
		}

		RegisterTelemetryIntegration(integration1)
		RegisterTelemetryIntegration(integration2)

		result := GetGlobalTelemetryIntegrations()
		assert.Equal(t, 2, len(result))

		// Verify order
		_ = result[0].OnStart(nil)
		_ = result[1].OnFinish(nil)
		assert.Equal(t, []string{"first", "second"}, calls)
	})
}

func TestGetGlobalTelemetryIntegrations(t *testing.T) {
	t.Run("returns an empty slice when no integrations are registered", func(t *testing.T) {
		ResetGlobalTelemetryIntegrations()

		result := GetGlobalTelemetryIntegrations()
		assert.Equal(t, []TelemetryIntegration{}, result)
	})
}
