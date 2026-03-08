// Ported from: packages/ai/src/telemetry/select-temetry-attributes.ts
// (note: the original filename has a typo "temetry" — Go file uses the corrected spelling)
package telemetry

// ResolvableAttributeValue is a function that returns an AttributeValue.
type ResolvableAttributeValue func() (AttributeValue, error)

// TelemetryAttributeEntry represents a single entry in the attributes map.
// It can be a plain AttributeValue, an input resolver, or an output resolver.
type TelemetryAttributeEntry struct {
	// Value is a plain attribute value. Mutually exclusive with Input and Output.
	Value AttributeValue
	// Input is a resolvable value that should only be recorded if RecordInputs is enabled.
	Input ResolvableAttributeValue
	// Output is a resolvable value that should only be recorded if RecordOutputs is enabled.
	Output ResolvableAttributeValue
	// IsInput indicates this entry is an input type.
	IsInput bool
	// IsOutput indicates this entry is an output type.
	IsOutput bool
}

// NewPlainAttribute creates a TelemetryAttributeEntry with a plain value.
func NewPlainAttribute(value AttributeValue) TelemetryAttributeEntry {
	return TelemetryAttributeEntry{Value: value}
}

// NewInputAttribute creates a TelemetryAttributeEntry for an input resolver.
func NewInputAttribute(fn ResolvableAttributeValue) TelemetryAttributeEntry {
	return TelemetryAttributeEntry{Input: fn, IsInput: true}
}

// NewOutputAttribute creates a TelemetryAttributeEntry for an output resolver.
func NewOutputAttribute(fn ResolvableAttributeValue) TelemetryAttributeEntry {
	return TelemetryAttributeEntry{Output: fn, IsOutput: true}
}

// SelectTelemetryAttributes selects and resolves telemetry attributes based
// on telemetry settings. When telemetry is disabled, returns an empty map
// to avoid serialization overhead.
func SelectTelemetryAttributes(
	telemetry *TelemetrySettings,
	attributes map[string]TelemetryAttributeEntry,
) (Attributes, error) {
	// when telemetry is disabled, return an empty map to avoid serialization overhead
	if telemetry == nil || telemetry.IsEnabled == nil || !*telemetry.IsEnabled {
		return Attributes{}, nil
	}

	result := Attributes{}

	for key, entry := range attributes {
		// input value, check if it should be recorded
		if entry.IsInput && entry.Input != nil {
			// default to true
			if telemetry.RecordInputs != nil && !*telemetry.RecordInputs {
				continue
			}

			val, err := entry.Input()
			if err != nil {
				return nil, err
			}
			if val != nil {
				result[key] = val
			}
			continue
		}

		// output value, check if it should be recorded
		if entry.IsOutput && entry.Output != nil {
			// default to true
			if telemetry.RecordOutputs != nil && !*telemetry.RecordOutputs {
				continue
			}

			val, err := entry.Output()
			if err != nil {
				return nil, err
			}
			if val != nil {
				result[key] = val
			}
			continue
		}

		// value is a plain attribute value
		if entry.Value != nil {
			result[key] = entry.Value
		}
	}

	return result, nil
}
