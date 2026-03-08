// Ported from: packages/ai/src/telemetry/get-global-telemetry-integration.test.ts
package telemetry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGlobalTelemetryIntegration(t *testing.T) {
	dummyEvent := struct{}{}

	setup := func() {
		ResetGlobalTelemetryIntegrations()
	}

	t.Run("returns no-op listeners when integrations is nil and no global integrations", func(t *testing.T) {
		setup()

		listeners := GetGlobalTelemetryIntegration()(nil)

		assert.NotNil(t, listeners.OnStart)
		assert.NotNil(t, listeners.OnStepStart)
		assert.NotNil(t, listeners.OnToolCallStart)
		assert.NotNil(t, listeners.OnToolCallFinish)
		assert.NotNil(t, listeners.OnStepFinish)
		assert.NotNil(t, listeners.OnFinish)

		err := listeners.OnStart(dummyEvent)
		assert.NoError(t, err)
	})

	t.Run("accepts a single integration", func(t *testing.T) {
		setup()

		called := false
		integration := TelemetryIntegration{
			OnStart: func(event interface{}) error {
				called = true
				return nil
			},
		}

		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{integration})

		assert.NotNil(t, listeners.OnStart)
		_ = listeners.OnStart(dummyEvent)
		assert.True(t, called)
	})

	t.Run("accepts an array of integrations", func(t *testing.T) {
		setup()

		startCalled := false
		finishCalled := false
		integration1 := TelemetryIntegration{
			OnStart: func(event interface{}) error { startCalled = true; return nil },
		}
		integration2 := TelemetryIntegration{
			OnFinish: func(event interface{}) error { finishCalled = true; return nil },
		}

		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{integration1, integration2})

		assert.NotNil(t, listeners.OnStart)
		assert.NotNil(t, listeners.OnFinish)

		_ = listeners.OnStart(dummyEvent)
		_ = listeners.OnFinish(dummyEvent)
		assert.True(t, startCalled)
		assert.True(t, finishCalled)
	})

	t.Run("returns no-op for a lifecycle method no integration implements", func(t *testing.T) {
		setup()

		integration := TelemetryIntegration{
			OnStart: func(event interface{}) error { return nil },
		}

		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{integration})

		assert.NotNil(t, listeners.OnToolCallStart)
		assert.NotNil(t, listeners.OnToolCallFinish)
		assert.NotNil(t, listeners.OnStepFinish)
		assert.NotNil(t, listeners.OnFinish)

		err := listeners.OnToolCallStart(dummyEvent)
		assert.NoError(t, err)
	})

	t.Run("broadcasts an event to all integrations that implement the method", func(t *testing.T) {
		setup()

		calls := make([]string, 0)
		integration1 := TelemetryIntegration{
			OnStart: func(event interface{}) error { calls = append(calls, "first"); return nil },
		}
		integration2 := TelemetryIntegration{
			OnStart: func(event interface{}) error { calls = append(calls, "second"); return nil },
		}

		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{integration1, integration2})
		_ = listeners.OnStart(dummyEvent)

		assert.Equal(t, []string{"first", "second"}, calls)
	})

	t.Run("calls integrations in order", func(t *testing.T) {
		setup()

		callOrder := make([]string, 0)
		integration1 := TelemetryIntegration{
			OnFinish: func(event interface{}) error {
				callOrder = append(callOrder, "first")
				return nil
			},
		}
		integration2 := TelemetryIntegration{
			OnFinish: func(event interface{}) error {
				callOrder = append(callOrder, "second")
				return nil
			},
		}

		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{integration1, integration2})
		_ = listeners.OnFinish(dummyEvent)

		assert.Equal(t, []string{"first", "second"}, callOrder)
	})

	t.Run("skips integrations that do not implement the method", func(t *testing.T) {
		setup()

		calls := 0
		integration1 := TelemetryIntegration{
			OnStart: func(event interface{}) error { calls++; return nil },
		}
		integration2 := TelemetryIntegration{} // no OnStart

		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{integration1, integration2})
		_ = listeners.OnStart(dummyEvent)

		assert.Equal(t, 1, calls)
	})

	t.Run("swallows errors from individual integrations without affecting others", func(t *testing.T) {
		setup()

		secondCalled := false
		integration1 := TelemetryIntegration{
			OnStart: func(event interface{}) error { return errors.New("boom") },
		}
		integration2 := TelemetryIntegration{
			OnStart: func(event interface{}) error { secondCalled = true; return nil },
		}

		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{integration1, integration2})
		err := listeners.OnStart(dummyEvent)

		assert.NoError(t, err)
		assert.True(t, secondCalled)
	})

	t.Run("swallows panics from integrations", func(t *testing.T) {
		setup()

		integration := TelemetryIntegration{
			OnStart: func(event interface{}) error { panic("sync boom") },
		}

		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{integration})

		err := listeners.OnStart(dummyEvent)
		assert.NoError(t, err)
	})

	t.Run("works with all lifecycle methods", func(t *testing.T) {
		setup()

		calls := make([]string, 0)
		integration := TelemetryIntegration{
			OnStart:          func(event interface{}) error { calls = append(calls, "start"); return nil },
			OnStepStart:      func(event interface{}) error { calls = append(calls, "stepStart"); return nil },
			OnToolCallStart:  func(event interface{}) error { calls = append(calls, "toolCallStart"); return nil },
			OnToolCallFinish: func(event interface{}) error { calls = append(calls, "toolCallFinish"); return nil },
			OnStepFinish:     func(event interface{}) error { calls = append(calls, "stepFinish"); return nil },
			OnFinish:         func(event interface{}) error { calls = append(calls, "finish"); return nil },
		}

		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{integration})

		_ = listeners.OnStart(dummyEvent)
		_ = listeners.OnStepStart(dummyEvent)
		_ = listeners.OnToolCallStart(dummyEvent)
		_ = listeners.OnToolCallFinish(dummyEvent)
		_ = listeners.OnStepFinish(dummyEvent)
		_ = listeners.OnFinish(dummyEvent)

		assert.Equal(t, []string{
			"start", "stepStart", "toolCallStart",
			"toolCallFinish", "stepFinish", "finish",
		}, calls)
	})

	t.Run("handles an empty slice of integrations", func(t *testing.T) {
		setup()

		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{})

		assert.NotNil(t, listeners.OnStart)
		assert.NotNil(t, listeners.OnFinish)

		err := listeners.OnStart(dummyEvent)
		assert.NoError(t, err)
	})

	t.Run("global integration merging", func(t *testing.T) {
		t.Run("includes globally registered integrations when no local integrations are provided", func(t *testing.T) {
			setup()

			called := false
			RegisterTelemetryIntegration(TelemetryIntegration{
				OnStart: func(event interface{}) error { called = true; return nil },
			})

			listeners := GetGlobalTelemetryIntegration()(nil)
			_ = listeners.OnStart(dummyEvent)

			assert.True(t, called)
		})

		t.Run("merges global and local integrations", func(t *testing.T) {
			setup()

			globalCalled := false
			localCalled := false

			RegisterTelemetryIntegration(TelemetryIntegration{
				OnStart: func(event interface{}) error { globalCalled = true; return nil },
			})

			listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{
				{OnStart: func(event interface{}) error { localCalled = true; return nil }},
			})
			_ = listeners.OnStart(dummyEvent)

			assert.True(t, globalCalled)
			assert.True(t, localCalled)
		})

		t.Run("calls global integrations before local integrations", func(t *testing.T) {
			setup()

			callOrder := make([]string, 0)

			RegisterTelemetryIntegration(TelemetryIntegration{
				OnFinish: func(event interface{}) error {
					callOrder = append(callOrder, "global")
					return nil
				},
			})

			listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{
				{OnFinish: func(event interface{}) error {
					callOrder = append(callOrder, "local")
					return nil
				}},
			})
			_ = listeners.OnFinish(dummyEvent)

			assert.Equal(t, []string{"global", "local"}, callOrder)
		})

		t.Run("global integrations work with local integration arrays", func(t *testing.T) {
			setup()

			calls := make([]string, 0)

			RegisterTelemetryIntegration(TelemetryIntegration{
				OnStart: func(event interface{}) error { calls = append(calls, "global"); return nil },
			})

			listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{
				{OnStart: func(event interface{}) error { calls = append(calls, "local1"); return nil }},
				{OnStart: func(event interface{}) error { calls = append(calls, "local2"); return nil }},
			})
			_ = listeners.OnStart(dummyEvent)

			assert.Equal(t, []string{"global", "local1", "local2"}, calls)
		})

		t.Run("errors in global integrations do not affect local integrations", func(t *testing.T) {
			setup()

			localCalled := false

			RegisterTelemetryIntegration(TelemetryIntegration{
				OnStart: func(event interface{}) error { return errors.New("global boom") },
			})

			listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{
				{OnStart: func(event interface{}) error { localCalled = true; return nil }},
			})
			_ = listeners.OnStart(dummyEvent)

			assert.True(t, localCalled)
		})
	})
}

func TestBindTelemetryIntegration(t *testing.T) {
	t.Run("preserves function references", func(t *testing.T) {
		called := false
		integration := TelemetryIntegration{
			OnStart: func(event interface{}) error { called = true; return nil },
		}

		bound := BindTelemetryIntegration(integration)

		_ = bound.OnStart(nil)
		assert.True(t, called)
	})

	t.Run("returns nil for methods the integration does not implement", func(t *testing.T) {
		integration := TelemetryIntegration{
			OnStart: func(event interface{}) error { return nil },
		}
		bound := BindTelemetryIntegration(integration)

		assert.NotNil(t, bound.OnStart)
		assert.Nil(t, bound.OnStepStart)
		assert.Nil(t, bound.OnToolCallStart)
		assert.Nil(t, bound.OnToolCallFinish)
		assert.Nil(t, bound.OnStepFinish)
		assert.Nil(t, bound.OnFinish)
	})

	t.Run("bound integration works correctly with GetGlobalTelemetryIntegration", func(t *testing.T) {
		ResetGlobalTelemetryIntegrations()

		calls := make([]string, 0)
		integration := TelemetryIntegration{
			OnStart:  func(event interface{}) error { calls = append(calls, "start"); return nil },
			OnFinish: func(event interface{}) error { calls = append(calls, "finish"); return nil },
		}

		bound := BindTelemetryIntegration(integration)
		listeners := GetGlobalTelemetryIntegration()([]TelemetryIntegration{bound})

		_ = listeners.OnStart(nil)
		_ = listeners.OnFinish(nil)

		assert.Equal(t, []string{"start", "finish"}, calls)
	})
}
