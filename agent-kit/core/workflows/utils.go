// Ported from: packages/core/src/workflows/utils.ts
package workflows

import (
	"fmt"
	"strings"
	"time"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// ---------------------------------------------------------------------------
// Zod-like validation helpers
// ---------------------------------------------------------------------------

// GetZodErrors extracts validation issues from a schema validation error.
// In Go, schemas use the SchemaWithValidation interface; this helper
// mirrors the TS getZodErrors that extracts ZodIssue[].
// Since Go doesn't have Zod, we return the error message directly.
func GetZodErrors(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ValidateStepInputResult is the result of ValidateStepInput.
type ValidateStepInputResult struct {
	InputData       any
	ValidationError error
}

// ValidateStepInput validates step input against the step's input schema.
func ValidateStepInput(prevOutput any, step *Step, validateInputs bool) ValidateStepInputResult {
	inputData := prevOutput
	var validationError error

	if validateInputs && step.InputSchema != nil {
		result, err := step.InputSchema.SafeParse(inputData)
		if err != nil {
			validationError = mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "WORKFLOW_STEP_INPUT_VALIDATION_FAILED",
				Domain:   mastraerror.ErrorDomainMastraWorkflow,
				Category: mastraerror.ErrorCategoryUser,
				Text:     "Step input validation failed: " + err.Error(),
			})
		} else if result != nil && result.Success {
			if result.Data != nil {
				inputData = result.Data
			}
		}
	}

	return ValidateStepInputResult{
		InputData:       inputData,
		ValidationError: validationError,
	}
}

// ValidateStepResumeDataResult is the result of ValidateStepResumeData.
type ValidateStepResumeDataResult struct {
	ResumeData      any
	ValidationError error
}

// ValidateStepResumeData validates resume data against the step's resume schema.
func ValidateStepResumeData(resumeData any, step *Step) ValidateStepResumeDataResult {
	if resumeData == nil {
		return ValidateStepResumeDataResult{}
	}

	var validationError error

	if step.ResumeSchema != nil {
		result, err := step.ResumeSchema.SafeParse(resumeData)
		if err != nil {
			validationError = mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "WORKFLOW_STEP_RESUME_DATA_VALIDATION_FAILED",
				Domain:   mastraerror.ErrorDomainMastraWorkflow,
				Category: mastraerror.ErrorCategoryUser,
				Text:     "Step resume data validation failed: " + err.Error(),
			})
		} else if result != nil && result.Success {
			resumeData = result.Data
		}
	}

	return ValidateStepResumeDataResult{
		ResumeData:      resumeData,
		ValidationError: validationError,
	}
}

// ValidateStepSuspendDataResult is the result of ValidateStepSuspendData.
type ValidateStepSuspendDataResult struct {
	SuspendData     any
	ValidationError error
}

// ValidateStepSuspendData validates suspend data against the step's suspend schema.
func ValidateStepSuspendData(suspendData any, step *Step, validateInputs bool) ValidateStepSuspendDataResult {
	if suspendData == nil {
		return ValidateStepSuspendDataResult{}
	}

	var validationError error

	if step.SuspendSchema != nil && validateInputs {
		result, err := step.SuspendSchema.SafeParse(suspendData)
		if err != nil {
			validationError = mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "WORKFLOW_STEP_SUSPEND_DATA_VALIDATION_FAILED",
				Domain:   mastraerror.ErrorDomainMastraWorkflow,
				Category: mastraerror.ErrorCategoryUser,
				Text:     "Step suspend data validation failed: " + err.Error(),
			})
		} else if result != nil && result.Success {
			suspendData = result.Data
		}
	}

	return ValidateStepSuspendDataResult{
		SuspendData:     suspendData,
		ValidationError: validationError,
	}
}

// ValidateStepStateDataResult is the result of ValidateStepStateData.
type ValidateStepStateDataResult struct {
	StateData       any
	ValidationError error
}

// ValidateStepStateData validates state data against the step's state schema.
func ValidateStepStateData(stateData any, step *Step, validateInputs bool) ValidateStepStateDataResult {
	if stateData == nil {
		return ValidateStepStateDataResult{}
	}

	var validationError error

	if step.StateSchema != nil && validateInputs {
		result, err := step.StateSchema.SafeParse(stateData)
		if err != nil {
			validationError = fmt.Errorf("step state data validation failed: %w", err)
		} else if result != nil && result.Success {
			stateData = result.Data
		}
	}

	return ValidateStepStateDataResult{
		StateData:       stateData,
		ValidationError: validationError,
	}
}

// ValidateStepRequestContextResult is the result of ValidateStepRequestContext.
type ValidateStepRequestContextResult struct {
	ValidationError error
}

// ValidateStepRequestContext validates the request context against the step's request context schema.
func ValidateStepRequestContext(requestContext map[string]any, step *Step, validateInputs bool) ValidateStepRequestContextResult {
	var validationError error

	if step.RequestContextSchema != nil && validateInputs {
		err := step.RequestContextSchema.Validate(requestContext)
		if err != nil {
			validationError = mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "WORKFLOW_STEP_REQUEST_CONTEXT_VALIDATION_FAILED",
				Domain:   mastraerror.ErrorDomainMastraWorkflow,
				Category: mastraerror.ErrorCategoryUser,
				Text:     fmt.Sprintf("Step request context validation failed for step '%s': %s", step.ID, err.Error()),
			})
		}
	}

	return ValidateStepRequestContextResult{
		ValidationError: validationError,
	}
}

// GetResumeLabelsByStepID returns resume labels filtered by step ID.
func GetResumeLabelsByStepID(resumeLabels map[string]ResumeLabel, stepID string) map[string]ResumeLabel {
	result := make(map[string]ResumeLabel)
	for key, value := range resumeLabels {
		if value.StepID == stepID {
			result[key] = value
		}
	}
	return result
}

// RunCountDeprecationMessage is the deprecation message for the runCount field.
const RunCountDeprecationMessage = "Warning: 'runCount' is deprecated and will be removed on November 4th, 2025. Please use 'retryCount' instead."

// shownWarnings tracks which deprecation warnings have been shown globally to avoid spam.
var shownWarnings = map[string]bool{}

// LogDeprecationWarning logs a deprecation warning if it hasn't been shown before.
// In Go we don't have JS Proxy, so callers must explicitly call this.
func LogDeprecationWarning(paramName string, deprecationMessage string, log logger.IMastraLogger) {
	if shownWarnings[paramName] {
		return
	}
	shownWarnings[paramName] = true
	if log != nil {
		log.Warn(deprecationMessage)
	}
}

// GetStepIDs returns the step IDs for a given StepFlowEntry.
func GetStepIDs(entry *StepFlowEntry) []string {
	if entry == nil {
		return nil
	}
	switch entry.Type {
	case StepFlowEntryTypeStep, StepFlowEntryTypeForeach, StepFlowEntryTypeLoop:
		if entry.Step != nil {
			return []string{entry.Step.ID}
		}
		return nil
	case StepFlowEntryTypeParallel, StepFlowEntryTypeConditional:
		ids := make([]string, 0, len(entry.Steps))
		for _, s := range entry.Steps {
			if s.Step != nil {
				ids = append(ids, s.Step.ID)
			}
		}
		return ids
	case StepFlowEntryTypeSleep, StepFlowEntryTypeSleepUntil:
		if entry.ID != "" {
			return []string{entry.ID}
		}
		return nil
	}
	return nil
}

// CreateTimeTravelExecutionParams creates TimeTravelExecutionParams from the given parameters.
func CreateTimeTravelExecutionParams(params CreateTimeTravelParams) (*TimeTravelExecutionParams, error) {
	steps := params.Steps
	inputData := params.InputData
	resumeData := params.ResumeData
	ctx := params.Context
	snapshot := params.Snapshot
	initialState := params.InitialState
	graph := params.Graph
	perStep := params.PerStep

	if len(steps) == 0 {
		return nil, fmt.Errorf("time travel target step not found: no steps provided")
	}
	firstStepID := steps[0]

	var executionPath []int
	stepResults := make(map[string]StepResult)
	snapshotContext := snapshot.Context
	if snapshotContext == nil {
		snapshotContext = make(map[string]any)
	}

	for index, entry := range graph.Steps {
		currentExecPathLength := len(executionPath)
		// If there is resumeData, steps down the graph until the suspended step
		// will have stepResult info to use
		if currentExecPathLength > 0 && resumeData == nil {
			break
		}

		stepIDs := GetStepIDs(&entry)
		if containsString(stepIDs, firstStepID) {
			var innerExecutionPath []int
			if len(stepIDs) > 1 {
				for i, s := range stepIDs {
					if s == firstStepID {
						innerExecutionPath = []int{i}
						break
					}
				}
			}
			executionPath = append([]int{index}, innerExecutionPath...)
		}

		// Get previous step payload
		var stepPayload any
		if index > 0 {
			prevStep := graph.Steps[index-1]
			prevStepIDs := GetStepIDs(&prevStep)
			if len(prevStepIDs) > 0 {
				if len(prevStepIDs) == 1 {
					if sr, ok := stepResults[prevStepIDs[0]]; ok {
						stepPayload = sr.Output
					}
					if stepPayload == nil {
						stepPayload = map[string]any{}
					}
				} else {
					m := make(map[string]any)
					for _, sid := range prevStepIDs {
						if sr, ok := stepResults[sid]; ok {
							m[sid] = sr.Output
						} else {
							m[sid] = map[string]any{}
						}
					}
					stepPayload = m
				}
			}
		}

		// Set input stepResult
		if index == 0 && containsString(stepIDs, firstStepID) {
			var inputVal any
			if ctx != nil {
				if e, ok := ctx[firstStepID]; ok {
					inputVal = e.Payload
				}
			}
			if inputVal == nil {
				inputVal = inputData
			}
			if inputVal == nil {
				if v, ok := snapshotContext["input"]; ok {
					inputVal = v
				}
			}
			stepResults["input"] = StepResult{Status: StepStatusSuccess, Output: inputVal}
		} else if index == 0 {
			var inputVal any
			for _, sid := range stepIDs {
				if inputVal != nil {
					break
				}
				if ctx != nil {
					if e, ok := ctx[sid]; ok {
						inputVal = e.Payload
						break
					}
				}
				if sc, ok := snapshotContext[sid]; ok {
					if scm, ok := sc.(map[string]any); ok {
						inputVal = scm["payload"]
					}
				}
			}
			if inputVal == nil {
				if v, ok := snapshotContext["input"]; ok {
					inputVal = v
				}
			}
			if inputVal == nil {
				inputVal = map[string]any{}
			}
			stepResults["input"] = StepResult{Status: StepStatusSuccess, Output: inputVal}
		}

		// Check if inputData should be used as stepOutput
		var stepOutput any
		if index+1 < len(graph.Steps) {
			nextStep := graph.Steps[index+1]
			nextStepIDs := GetStepIDs(&nextStep)
			if len(nextStepIDs) > 0 && inputData != nil &&
				containsString(nextStepIDs, firstStepID) && len(steps) == 1 {
				stepOutput = inputData
			}
		}

		for _, stepID := range stepIDs {
			var stepContext map[string]any
			if ctx != nil {
				if e, ok := ctx[stepID]; ok {
					stepContext = map[string]any{
						"status":         string(e.Status),
						"payload":        e.Payload,
						"output":         e.Output,
						"resumePayload":  e.ResumePayload,
						"suspendPayload": e.SuspendPayload,
						"suspendOutput":  e.SuspendOutput,
						"startedAt":      e.StartedAt,
						"endedAt":        e.EndedAt,
						"suspendedAt":    e.SuspendedAt,
						"resumedAt":      e.ResumedAt,
					}
				}
			}
			if stepContext == nil {
				if sc, ok := snapshotContext[stepID]; ok {
					if scm, ok := sc.(map[string]any); ok {
						stepContext = scm
					}
				}
			}

			defaultStepStatus := WorkflowStepStatus("success")
			if containsString(steps, stepID) {
				defaultStepStatus = "running"
			}

			status := defaultStepStatus
			if stepContext != nil {
				if s, ok := stepContext["status"].(string); ok {
					if s == "failed" || s == "canceled" {
						status = defaultStepStatus
					} else {
						status = WorkflowStepStatus(s)
					}
				}
			}

			isCompleteStatus := status == "success" || status == "failed" || status == "canceled"

			now := time.Now().UnixMilli()

			var payload any
			if ctx != nil {
				if e, ok := ctx[stepID]; ok {
					payload = e.Payload
				}
			}
			if payload == nil {
				payload = stepPayload
			}
			if payload == nil {
				if stepContext != nil {
					payload = stepContext["payload"]
				}
			}
			if payload == nil {
				payload = map[string]any{}
			}

			var output any
			if isCompleteStatus {
				if ctx != nil {
					if e, ok := ctx[stepID]; ok {
						output = e.Output
					}
				}
				if output == nil {
					output = stepOutput
				}
				if output == nil {
					if stepContext != nil {
						output = stepContext["output"]
					}
				}
				if output == nil {
					output = map[string]any{}
				}
			}

			startedAt := now
			if stepContext != nil {
				if sa, ok := stepContext["startedAt"].(int64); ok {
					startedAt = sa
				} else if sa, ok := stepContext["startedAt"].(float64); ok {
					startedAt = int64(sa)
				}
			}

			var endedAt int64
			if isCompleteStatus {
				endedAt = now
				if stepContext != nil {
					if ea, ok := stepContext["endedAt"].(int64); ok {
						endedAt = ea
					} else if ea, ok := stepContext["endedAt"].(float64); ok {
						endedAt = int64(ea)
					}
				}
			}

			execPathLengthToUse := currentExecPathLength
			if perStep {
				execPathLengthToUse = len(executionPath)
			}

			// If the step is after the timeTravelled step in the graph and
			// it doesn't exist in the snapshot, or it exists but is not suspended,
			// we don't need to set stepResult for it.
			skipResult := false
			if execPathLengthToUse > 0 && !containsString(steps, stepID) {
				_, hasCtx := ctx[stepID]
			hasCtx = ctx != nil && hasCtx
				if !hasCtx {
					// Check if stepContext exists and is suspended
					scStatus := ""
					if stepContext != nil {
						if s, ok := stepContext["status"].(string); ok {
							scStatus = s
						}
					}
					if stepContext == nil || scStatus != "suspended" {
						skipResult = true
					}
				}
			}

			if !skipResult {
				sr := StepResult{
					Status:    status,
					Payload:   payload,
					Output:    output,
					StartedAt: startedAt,
					EndedAt:   endedAt,
				}
				if stepContext != nil {
					sr.ResumePayload = stepContext["resumePayload"]
					sr.SuspendPayload = stepContext["suspendPayload"]
					sr.SuspendOutput = stepContext["suspendOutput"]
					if sa, ok := stepContext["suspendedAt"].(*int64); ok {
						sr.SuspendedAt = sa
					}
					if ra, ok := stepContext["resumedAt"].(*int64); ok {
						sr.ResumedAt = ra
					}
				}
				stepResults[stepID] = sr
			}
		}
	}

	if len(executionPath) == 0 {
		return nil, fmt.Errorf(
			"time travel target step not found in execution graph: '%s'. Verify the step id/path",
			strings.Join(steps, "."),
		)
	}

	snapshotValue := snapshot.Value
	state := initialState
	if state == nil {
		if len(snapshotValue) > 0 {
			state = make(map[string]any)
			for k, v := range snapshotValue {
				state[k] = v
			}
		} else {
			state = make(map[string]any)
		}
	}

	return &TimeTravelExecutionParams{
		InputData:         inputData,
		ExecutionPath:     executionPath,
		Steps:             steps,
		StepResults:       stepResults,
		NestedStepResults: params.NestedStepsContext,
		State:             state,
		ResumeData:        resumeData,
		StepExecutionPath: snapshot.StepExecutionPath,
	}, nil
}

// CreateTimeTravelParams holds parameters for CreateTimeTravelExecutionParams.
type CreateTimeTravelParams struct {
	Steps              []string
	InputData          any
	ResumeData         any
	Context            TimeTravelContext
	NestedStepsContext map[string]map[string]StepResult
	Snapshot           WorkflowRunState
	InitialState       map[string]any
	Graph              ExecutionGraph
	PerStep            bool
}

// HydrateSerializedStepErrors re-hydrates serialized errors in step results back into
// proper error instances. Useful when errors have been serialized through an event system.
func HydrateSerializedStepErrors(steps map[string]any) map[string]any {
	if steps == nil {
		return steps
	}
	for key, step := range steps {
		if m, ok := step.(map[string]any); ok {
			if m["status"] == "failed" {
				if errVal, hasErr := m["error"]; hasErr && errVal != nil {
					if _, isErr := errVal.(error); !isErr {
						m["error"] = fmt.Errorf("%v", errVal)
					}
				}
			}
			steps[key] = m
		}
	}
	return steps
}

// cleanSingleResult removes internal properties from a single step result.
func cleanSingleResult(result map[string]any) map[string]any {
	cleaned := make(map[string]any)
	for k, v := range result {
		if k == "__state" {
			continue
		}
		if k == "metadata" {
			if m, ok := v.(map[string]any); ok {
				userMetadata := make(map[string]any)
				for mk, mv := range m {
					if mk != "nestedRunId" {
						userMetadata[mk] = mv
					}
				}
				if len(userMetadata) > 0 {
					cleaned["metadata"] = userMetadata
				}
				continue
			}
		}
		cleaned[k] = v
	}
	return cleaned
}

// CleanStepResult cleans step result data by removing internal properties at known structural levels.
// Removes __state properties and nestedRunId from metadata objects.
func CleanStepResult(stepResult any) any {
	if stepResult == nil {
		return stepResult
	}

	// Handle arrays (forEach iteration results)
	if arr, ok := stepResult.([]any); ok {
		cleaned := make([]any, len(arr))
		for i, item := range arr {
			if m, ok := item.(map[string]any); ok {
				cleaned[i] = cleanSingleResult(m)
			} else {
				cleaned[i] = item
			}
		}
		return cleaned
	}

	result, ok := stepResult.(map[string]any)
	if !ok {
		return stepResult
	}

	cleaned := cleanSingleResult(result)

	// If output is an array, clean each iteration result
	if output, ok := cleaned["output"].([]any); ok {
		cleanedOutput := make([]any, len(output))
		for i, item := range output {
			if m, ok := item.(map[string]any); ok {
				cleanedOutput[i] = cleanSingleResult(m)
			} else {
				cleanedOutput[i] = item
			}
		}
		cleaned["output"] = cleanedOutput
	}

	return cleaned
}

// RemoveUndefinedValues removes nil values from a map.
// In Go, nil is the closest equivalent to JS undefined.
func RemoveUndefinedValues(m map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		if v != nil {
			result[k] = v
		}
	}
	return result
}

// containsString checks if a string slice contains a given string.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
