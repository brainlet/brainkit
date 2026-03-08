// Ported from: packages/ai/src/generate-text/prepare-step.ts
package generatetext

// PrepareStepFunction is a function that provides different settings for a step.
type PrepareStepFunction func(options PrepareStepOptions) (*PrepareStepResult, error)

// PrepareStepOptions are the options passed to a PrepareStepFunction.
type PrepareStepOptions struct {
	// Steps are the steps that have been executed so far.
	Steps []StepResult

	// StepNumber is the number of the step that is being executed.
	StepNumber int

	// Model is the model instance being used for this step.
	Model LanguageModel

	// Messages are the messages that will be sent to the model for the current step.
	Messages []ModelMessage

	// ExperimentalContext is the context passed via the experimental_context setting.
	ExperimentalContext interface{}
}

// PrepareStepResult is the result returned by a PrepareStepFunction,
// allowing per-step overrides of model, tools, or messages.
type PrepareStepResult struct {
	// Model optionally overrides which model is used for this step.
	Model LanguageModel

	// ToolChoice optionally sets the tool choice for this step.
	ToolChoice *ToolChoice

	// ActiveTools optionally restricts which tools are available for this step.
	ActiveTools []string

	// System optionally overrides the system message(s) for this step.
	System interface{} // string | SystemModelMessage | []SystemModelMessage

	// Messages optionally overrides the full set of messages for this step.
	Messages []ModelMessage

	// ExperimentalContext optionally overrides the context for this step and all subsequent steps.
	ExperimentalContext interface{}

	// ProviderOptions optionally provides additional provider-specific options for this step.
	ProviderOptions ProviderOptions
}
