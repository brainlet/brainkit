// Ported from: packages/ai/src/generate-text/stop-condition.ts
package generatetext

// StopCondition is a function that determines whether generation should stop.
type StopCondition func(options StopConditionOptions) (bool, error)

// StopConditionOptions contains the context for evaluating a stop condition.
type StopConditionOptions struct {
	Steps []StepResult
}

// StepCountIs returns a StopCondition that stops when the given number of steps has been reached.
func StepCountIs(stepCount int) StopCondition {
	return func(options StopConditionOptions) (bool, error) {
		return len(options.Steps) == stepCount, nil
	}
}

// HasToolCall returns a StopCondition that stops when the last step contains
// a tool call with the given name.
func HasToolCall(toolName string) StopCondition {
	return func(options StopConditionOptions) (bool, error) {
		if len(options.Steps) == 0 {
			return false, nil
		}
		lastStep := options.Steps[len(options.Steps)-1]
		for _, tc := range lastStep.ToolCalls() {
			if tc.ToolName == toolName {
				return true, nil
			}
		}
		return false, nil
	}
}

// IsStopConditionMet evaluates all stop conditions and returns true if any is met.
func IsStopConditionMet(stopConditions []StopCondition, steps []StepResult) (bool, error) {
	for _, condition := range stopConditions {
		met, err := condition(StopConditionOptions{Steps: steps})
		if err != nil {
			return false, err
		}
		if met {
			return true, nil
		}
	}
	return false, nil
}
