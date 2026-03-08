// Ported from: packages/ai/src/generate-object/parse-and-validate-object-result.ts
package generateobject

import (
	"fmt"
)

// ParseAndValidateObjectResult parses a JSON string and validates it against
// the output strategy.
func ParseAndValidateObjectResult(result string, strategy OutputStrategy) (any, error) {
	parsed, err := ParseJSON(result)
	if err != nil {
		return nil, fmt.Errorf("no object generated: could not parse the response: %w", err)
	}

	validationResult := strategy.ValidateFinalResult(parsed)
	if !validationResult.Success {
		return nil, fmt.Errorf("no object generated: response did not match schema: %w", validationResult.Error)
	}

	return validationResult.Value, nil
}

// ParseAndValidateObjectResultWithRepair parses a JSON string, validates it,
// and optionally repairs it using the repairText function if parsing/validation fails.
func ParseAndValidateObjectResultWithRepair(
	result string,
	strategy OutputStrategy,
	repairText RepairTextFunc,
) (any, error) {
	value, err := ParseAndValidateObjectResult(result, strategy)
	if err == nil {
		return value, nil
	}

	if repairText != nil {
		repairedText, repairErr := repairText(result, err)
		if repairErr != nil {
			return nil, err // return original error
		}
		if repairedText == "" {
			return nil, err // repair returned empty, use original error
		}
		return ParseAndValidateObjectResult(repairedText, strategy)
	}

	return nil, err
}
