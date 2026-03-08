// Ported from: packages/ai/src/telemetry/select-temetry-attributes.test.ts
// (note: the original filename has a typo "temetry" which we preserve in this comment)
package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func boolPtr(b bool) *bool { return &b }

func TestSelectTelemetryAttributes(t *testing.T) {
	t.Run("should return an empty map when telemetry is disabled", func(t *testing.T) {
		result, err := SelectTelemetryAttributes(
			&TelemetrySettings{IsEnabled: boolPtr(false)},
			map[string]TelemetryAttributeEntry{
				"key": NewPlainAttribute("value"),
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, Attributes{}, result)
	})

	t.Run("should return an empty map when telemetry enablement is nil", func(t *testing.T) {
		result, err := SelectTelemetryAttributes(
			&TelemetrySettings{IsEnabled: nil},
			map[string]TelemetryAttributeEntry{
				"key": NewPlainAttribute("value"),
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, Attributes{}, result)
	})

	t.Run("should return attributes with simple values", func(t *testing.T) {
		result, err := SelectTelemetryAttributes(
			&TelemetrySettings{IsEnabled: boolPtr(true)},
			map[string]TelemetryAttributeEntry{
				"string":  NewPlainAttribute("value"),
				"number":  NewPlainAttribute(42),
				"boolean": NewPlainAttribute(true),
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, "value", result["string"])
		assert.Equal(t, 42, result["number"])
		assert.Equal(t, true, result["boolean"])
	})

	t.Run("should handle input functions when recordInputs is true", func(t *testing.T) {
		result, err := SelectTelemetryAttributes(
			&TelemetrySettings{IsEnabled: boolPtr(true), RecordInputs: boolPtr(true)},
			map[string]TelemetryAttributeEntry{
				"input": NewInputAttribute(func() (AttributeValue, error) { return "input value", nil }),
				"other": NewPlainAttribute("other value"),
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, "input value", result["input"])
		assert.Equal(t, "other value", result["other"])
	})

	t.Run("should not include input functions when recordInputs is false", func(t *testing.T) {
		result, err := SelectTelemetryAttributes(
			&TelemetrySettings{IsEnabled: boolPtr(true), RecordInputs: boolPtr(false)},
			map[string]TelemetryAttributeEntry{
				"input": NewInputAttribute(func() (AttributeValue, error) { return "input value", nil }),
				"other": NewPlainAttribute("other value"),
			},
		)
		assert.NoError(t, err)
		_, hasInput := result["input"]
		assert.False(t, hasInput)
		assert.Equal(t, "other value", result["other"])
	})

	t.Run("should handle output functions when recordOutputs is true", func(t *testing.T) {
		result, err := SelectTelemetryAttributes(
			&TelemetrySettings{IsEnabled: boolPtr(true), RecordOutputs: boolPtr(true)},
			map[string]TelemetryAttributeEntry{
				"output": NewOutputAttribute(func() (AttributeValue, error) { return "output value", nil }),
				"other":  NewPlainAttribute("other value"),
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, "output value", result["output"])
		assert.Equal(t, "other value", result["other"])
	})

	t.Run("should not include output functions when recordOutputs is false", func(t *testing.T) {
		result, err := SelectTelemetryAttributes(
			&TelemetrySettings{IsEnabled: boolPtr(true), RecordOutputs: boolPtr(false)},
			map[string]TelemetryAttributeEntry{
				"output": NewOutputAttribute(func() (AttributeValue, error) { return "output value", nil }),
				"other":  NewPlainAttribute("other value"),
			},
		)
		assert.NoError(t, err)
		_, hasOutput := result["output"]
		assert.False(t, hasOutput)
		assert.Equal(t, "other value", result["other"])
	})

	t.Run("should ignore nil values", func(t *testing.T) {
		result, err := SelectTelemetryAttributes(
			&TelemetrySettings{IsEnabled: boolPtr(true)},
			map[string]TelemetryAttributeEntry{
				"defined":   NewPlainAttribute("value"),
				"undefined": NewPlainAttribute(nil),
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, "value", result["defined"])
		_, hasUndefined := result["undefined"]
		assert.False(t, hasUndefined)
	})

	t.Run("should ignore input and output functions that return nil", func(t *testing.T) {
		result, err := SelectTelemetryAttributes(
			&TelemetrySettings{IsEnabled: boolPtr(true)},
			map[string]TelemetryAttributeEntry{
				"input":  NewInputAttribute(func() (AttributeValue, error) { return nil, nil }),
				"output": NewOutputAttribute(func() (AttributeValue, error) { return nil, nil }),
				"other":  NewPlainAttribute("value"),
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, Attributes{"other": "value"}, result)
	})

	t.Run("should handle mixed attribute types correctly", func(t *testing.T) {
		result, err := SelectTelemetryAttributes(
			&TelemetrySettings{IsEnabled: boolPtr(true)},
			map[string]TelemetryAttributeEntry{
				"simple":     NewPlainAttribute("value"),
				"input":      NewInputAttribute(func() (AttributeValue, error) { return "input value", nil }),
				"output":     NewOutputAttribute(func() (AttributeValue, error) { return "output value", nil }),
				"undefined":  NewPlainAttribute(nil),
				"input_null": NewInputAttribute(func() (AttributeValue, error) { return nil, nil }),
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, "value", result["simple"])
		assert.Equal(t, "input value", result["input"])
		assert.Equal(t, "output value", result["output"])
		_, hasUndefined := result["undefined"]
		assert.False(t, hasUndefined)
		_, hasInputNull := result["input_null"]
		assert.False(t, hasInputNull)
	})
}
