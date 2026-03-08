// Ported from: packages/core/src/evals/run/index.ts
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/evals"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ============================================================================
// Stub types — kept local to avoid circular import dependencies
// ============================================================================

// Agent is a stub for the agent.Agent type.
// CIRCULAR DEP: Cannot import agent — agent indirectly imports evals via
// core/hooks.go and workflows/types.go. The real agent.Agent has different
// method signatures (Generate takes agent-specific types, uses ID() not
// GetID(), etc.). Replacing requires both resolving the cycle and refactoring
// the eval runner.
type Agent interface {
	// Generate runs the agent on the given input.
	Generate(ctx context.Context, input any, opts map[string]any) (any, error)
	// GetModel returns the agent's model.
	GetModel(ctx context.Context) (any, error)
	// GetID returns the agent's ID.
	GetID() string
	// GetName returns the agent's name.
	GetName() string
	// GetMastraInstance returns the Mastra instance the agent is registered with.
	GetMastraInstance() any
}

// Workflow is a stub for the workflows.Workflow type.
// CIRCULAR DEP: Cannot import workflows — workflows/types.go imports evals.
// The real workflows.Workflow has different CreateRun signature and method set.
type Workflow interface {
	// CreateRun creates a new workflow run.
	CreateRun(ctx context.Context, opts map[string]any) (WorkflowRun, error)
	// GetID returns the workflow's ID.
	GetID() string
	// GetName returns the workflow's name.
	GetName() string
	// GetMastra returns the Mastra instance the workflow is registered with.
	GetMastra() any
}

// WorkflowRun is a stub for a workflow run instance.
// CIRCULAR DEP: Cannot import workflows — workflows/types.go imports evals.
// The real workflow run has different Start signature and return types.
type WorkflowRun interface {
	// Start starts the workflow run.
	Start(ctx context.Context, opts map[string]any) (*WorkflowResult, error)
}

// WorkflowResult is a stub for the workflow result.
// CIRCULAR DEP: Cannot import workflows — workflows/types.go imports evals.
// Simplified version of the real workflow run result.
type WorkflowResult struct {
	Status string         `json:"status"`
	Result any            `json:"result,omitempty"`
	Steps  map[string]any `json:"steps,omitempty"`
}

// StepResult is a stub for a workflow step result.
// CIRCULAR DEP: Cannot import workflows — workflows/types.go imports evals.
// Simplified version of the real workflow step result.
type StepResult struct {
	Status  string `json:"status"`
	Output  any    `json:"output,omitempty"`
	Payload any    `json:"payload,omitempty"`
}

// MastraStorage is a stub for the storage.MastraCompositeStore interface.
// CIRCULAR DEP: Cannot import storage — storage/types.go imports evals.
// The real MastraCompositeStore.GetStore takes storage.DomainName and returns
// domains.StorageDomain (not (any, error)). Signature differs.
type MastraStorage interface {
	// GetStore returns a domain-specific store.
	GetStore(domain string) (any, error)
}

// RequestContext is a stub for requestcontext.RequestContext.
// STRUCTURAL MISMATCH (not a circular dep — evals CAN import requestcontext).
// The real requestcontext.RequestContext is a struct with sync.RWMutex and
// methods (Get, Set, Has, Delete, etc.). This stub uses map[string]any for
// compatibility with the TS port where RequestContext was treated as a plain
// Record<string, any>. Replacing would require updating RunEvalsDataItem and
// all code that creates/accesses RequestContext values.
type RequestContext = map[string]any

// ============================================================================
// RunEvals Configuration Types
// ============================================================================

// RunEvalsDataItem represents a single data item for evaluation.
// Corresponds to TS: type RunEvalsDataItem<TTarget>
type RunEvalsDataItem struct {
	Input                any                           `json:"input"`
	GroundTruth          any                           `json:"groundTruth,omitempty"`
	RequestContext       RequestContext                `json:"requestContext,omitempty"`
	StartOptions         map[string]any                `json:"startOptions,omitempty"`
	ObservabilityContext *obstypes.ObservabilityContext `json:"-"`
}

// WorkflowScorerConfig configures scorers for workflow evaluation with
// optional per-step scorers.
// Corresponds to TS: type WorkflowScorerConfig
type WorkflowScorerConfig struct {
	Workflow []*evals.MastraScorer            `json:"workflow,omitempty"`
	Steps    map[string][]*evals.MastraScorer `json:"steps,omitempty"`
}

// RunEvalsResult is the result of running evaluations.
// Corresponds to TS: type RunEvalsResult
type RunEvalsResult struct {
	Scores  map[string]any `json:"scores"`
	Summary struct {
		TotalItems int `json:"totalItems"`
	} `json:"summary"`
}

// OnItemCompleteParams holds the parameters passed to the onItemComplete callback.
type OnItemCompleteParams struct {
	Item          RunEvalsDataItem
	TargetResult  any
	ScorerResults map[string]any
}

// RunEvalsConfig holds the configuration for RunEvals.
type RunEvalsConfig struct {
	Data           []RunEvalsDataItem
	Scorers        any // []*evals.MastraScorer or *WorkflowScorerConfig
	Target         any // Agent or Workflow
	TargetOptions  map[string]any
	OnItemComplete func(params OnItemCompleteParams) error
	Concurrency    int
}

// ============================================================================
// RunEvals — main entry point
// ============================================================================

// RunEvals executes scorers against a target (Agent or Workflow) for each
// data item, accumulates scores, and returns average results.
//
// Corresponds to TS: export async function runEvals(config: {...}): Promise<RunEvalsResult>
func RunEvals(ctx context.Context, config RunEvalsConfig) (*RunEvalsResult, error) {
	concurrency := config.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	if err := validateEvalsInputs(config.Data, config.Scorers, config.Target); err != nil {
		return nil, err
	}

	totalItems := 0
	scoreAccumulator := NewScoreAccumulator()

	// Get storage from target's Mastra instance if available.
	var storage MastraStorage
	mastra := getMastraFromTarget(config.Target)
	if mastra != nil {
		storage = getStorageFromMastra(mastra)
	}

	// Process data items with concurrency control.
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup

	for _, item := range config.Data {
		if firstErr != nil {
			break
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(item RunEvalsDataItem) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			targetResult, err := executeTarget(ctx, config.Target, item, config.TargetOptions)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			scorerResults, err := runScorers(ctx, config.Scorers, targetResult, item)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			scoreAccumulator.AddScores(scorerResults)
			totalItems++
			mu.Unlock()

			// Save scores to storage if available.
			if storage != nil {
				_ = saveScoresToStorage(ctx, storage, scorerResults, config.Target, item, mastra)
			}

			if config.OnItemComplete != nil {
				if err := config.OnItemComplete(OnItemCompleteParams{
					Item:          item,
					TargetResult:  targetResult,
					ScorerResults: scorerResults,
				}); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
			}
		}(item)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return &RunEvalsResult{
		Scores: scoreAccumulator.GetAverageScores(),
		Summary: struct {
			TotalItems int `json:"totalItems"`
		}{
			TotalItems: totalItems,
		},
	}, nil
}

// ============================================================================
// Validation
// ============================================================================

// validateEvalsInputs validates the inputs for RunEvals.
// Corresponds to TS: function validateEvalsInputs(...)
func validateEvalsInputs(data []RunEvalsDataItem, scorers any, target any) error {
	if len(data) == 0 {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			Domain:   mastraerror.ErrorDomainScorer,
			ID:       "RUN_EXPERIMENT_FAILED_NO_DATA_PROVIDED",
			Category: mastraerror.ErrorCategoryUser,
			Text:     "Failed to run experiment: Data array is empty",
		})
	}

	for i, item := range data {
		if item.Input == nil {
			return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				Domain:   mastraerror.ErrorDomainScorer,
				ID:       "INVALID_DATA_ITEM",
				Category: mastraerror.ErrorCategoryUser,
				Text:     fmt.Sprintf("Invalid data item at index %d: must have 'input' properties", i),
			})
		}
	}

	// Validate scorers based on type.
	switch s := scorers.(type) {
	case []*evals.MastraScorer:
		if len(s) == 0 {
			return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				Domain:   mastraerror.ErrorDomainScorer,
				ID:       "NO_SCORERS_PROVIDED",
				Category: mastraerror.ErrorCategoryUser,
				Text:     "At least one scorer must be provided",
			})
		}
	case *WorkflowScorerConfig:
		hasScorers := (s.Workflow != nil && len(s.Workflow) > 0) ||
			(s.Steps != nil && len(s.Steps) > 0)
		if !hasScorers {
			return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				Domain:   mastraerror.ErrorDomainScorer,
				ID:       "NO_SCORERS_PROVIDED",
				Category: mastraerror.ErrorCategoryUser,
				Text:     "At least one workflow or step scorer must be provided",
			})
		}
	default:
		// If target is not a Workflow and scorers is not an array, error.
		if _, isWorkflow := target.(Workflow); !isWorkflow {
			return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				Domain:   mastraerror.ErrorDomainScorer,
				ID:       "INVALID_AGENT_SCORERS",
				Category: mastraerror.ErrorCategoryUser,
				Text:     "Agent scorers must be an array of scorers",
			})
		}
	}

	return nil
}

// ============================================================================
// Target execution
// ============================================================================

// executeTarget runs the target (Agent or Workflow) on the given data item.
// Corresponds to TS: async function executeTarget(...)
func executeTarget(ctx context.Context, target any, item RunEvalsDataItem, targetOptions map[string]any) (any, error) {
	switch t := target.(type) {
	case Workflow:
		return executeWorkflow(ctx, t, item, targetOptions)
	case Agent:
		return executeAgent(ctx, t, item, targetOptions)
	default:
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			Domain:   mastraerror.ErrorDomainScorer,
			ID:       "RUN_EXPERIMENT_TARGET_FAILED_TO_GENERATE_RESULT",
			Category: mastraerror.ErrorCategoryUser,
			Text:     "Failed to run experiment: Unknown target type",
		})
	}
}

// executeWorkflow runs a workflow target.
// Corresponds to TS: async function executeWorkflow(...)
func executeWorkflow(ctx context.Context, target Workflow, item RunEvalsDataItem, targetOptions map[string]any) (any, error) {
	opts := make(map[string]any)
	if targetOptions != nil {
		for k, v := range targetOptions {
			opts[k] = v
		}
	}
	if item.StartOptions != nil {
		for k, v := range item.StartOptions {
			opts[k] = v
		}
	}
	opts["inputData"] = item.Input
	opts["requestContext"] = item.RequestContext
	opts["disableScorers"] = true

	run, err := target.CreateRun(ctx, map[string]any{"disableScorers": true})
	if err != nil {
		return nil, wrapTargetError(err, item)
	}

	workflowResult, err := run.Start(ctx, opts)
	if err != nil {
		return nil, wrapTargetError(err, item)
	}

	var output any
	if workflowResult.Status == "success" {
		output = workflowResult.Result
	}

	return map[string]any{
		"scoringData": map[string]any{
			"input":       item.Input,
			"output":      output,
			"stepResults": workflowResult.Steps,
		},
	}, nil
}

// executeAgent runs an agent target.
// Corresponds to TS: async function executeAgent(...)
func executeAgent(ctx context.Context, agent Agent, item RunEvalsDataItem, targetOptions map[string]any) (any, error) {
	opts := make(map[string]any)
	if targetOptions != nil {
		for k, v := range targetOptions {
			opts[k] = v
		}
	}
	opts["scorers"] = map[string]any{}
	opts["returnScorerData"] = true
	opts["requestContext"] = item.RequestContext

	result, err := agent.Generate(ctx, item.Input, opts)
	if err != nil {
		return nil, wrapTargetError(err, item)
	}

	return result, nil
}

// wrapTargetError wraps a target execution error in a MastraError.
func wrapTargetError(err error, item RunEvalsDataItem) error {
	itemJSON, _ := json.Marshal(item)
	return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		Domain:   mastraerror.ErrorDomainScorer,
		ID:       "RUN_EXPERIMENT_TARGET_FAILED_TO_GENERATE_RESULT",
		Category: mastraerror.ErrorCategoryUser,
		Text:     "Failed to run experiment: Error generating result from target",
		Details: map[string]any{
			"item":  string(itemJSON),
			"error": err.Error(),
		},
	})
}

// ============================================================================
// Scorer execution
// ============================================================================

// runScorers runs all scorers against the target result for a data item.
// Corresponds to TS: async function runScorers(...)
func runScorers(ctx context.Context, scorers any, targetResult any, item RunEvalsDataItem) (map[string]any, error) {
	scorerResults := make(map[string]any)

	switch s := scorers.(type) {
	case []*evals.MastraScorer:
		for _, scorer := range s {
			scoringData := extractScoringData(targetResult)
			score, err := scorer.Run(ctx, &evals.ScorerRun{
				Input:       scoringData["input"],
				Output:      scoringData["output"],
				GroundTruth: item.GroundTruth,
				RequestContext: item.RequestContext,
			})
			if err != nil {
				return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					Domain:   mastraerror.ErrorDomainScorer,
					ID:       "RUN_EXPERIMENT_SCORER_FAILED_TO_SCORE_RESULT",
					Category: mastraerror.ErrorCategoryUser,
					Text:     fmt.Sprintf("Failed to run experiment: Error running scorer %s", scorer.ID()),
					Details: map[string]any{
						"scorerId": scorer.ID(),
					},
				})
			}
			scorerResults[scorer.ID()] = score
		}

	case *WorkflowScorerConfig:
		// Handle workflow-level scorers.
		if s.Workflow != nil && len(s.Workflow) > 0 {
			workflowScorerResults := make(map[string]any)
			scoringData := extractScoringData(targetResult)
			for _, scorer := range s.Workflow {
				score, err := scorer.Run(ctx, &evals.ScorerRun{
					Input:          scoringData["input"],
					Output:         scoringData["output"],
					GroundTruth:    item.GroundTruth,
					RequestContext: item.RequestContext,
				})
				if err != nil {
					return nil, err
				}
				workflowScorerResults[scorer.ID()] = score
			}
			if len(workflowScorerResults) > 0 {
				scorerResults["workflow"] = workflowScorerResults
			}
		}

		// Handle step-level scorers.
		if s.Steps != nil && len(s.Steps) > 0 {
			stepScorerResults := make(map[string]any)
			scoringData := extractScoringData(targetResult)
			stepResults, _ := scoringData["stepResults"].(map[string]any)

			for stepID, stepScorers := range s.Steps {
				if stepResults == nil {
					continue
				}
				stepResult, ok := stepResults[stepID].(map[string]any)
				if !ok {
					continue
				}
				status, _ := stepResult["status"].(string)
				if status != "success" || stepResult["output"] == nil {
					continue
				}

				stepScoreResults := make(map[string]any)
				for _, scorer := range stepScorers {
					input := stepResult["payload"]
					if input == nil {
						input = scoringData["input"]
					}
					score, err := scorer.Run(ctx, &evals.ScorerRun{
						Input:          input,
						Output:         stepResult["output"],
						GroundTruth:    item.GroundTruth,
						RequestContext: item.RequestContext,
					})
					if err != nil {
						return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
							Domain:   mastraerror.ErrorDomainScorer,
							ID:       "RUN_EXPERIMENT_SCORER_FAILED_TO_SCORE_STEP_RESULT",
							Category: mastraerror.ErrorCategoryUser,
							Text:     fmt.Sprintf("Failed to run experiment: Error running scorer %s on step %s", scorer.ID(), stepID),
							Details: map[string]any{
								"scorerId": scorer.ID(),
								"stepId":   stepID,
							},
						})
					}
					stepScoreResults[scorer.ID()] = score
				}
				if len(stepScoreResults) > 0 {
					stepScorerResults[stepID] = stepScoreResults
				}
			}
			if len(stepScorerResults) > 0 {
				scorerResults["steps"] = stepScorerResults
			}
		}
	}

	return scorerResults, nil
}

// ============================================================================
// Score persistence
// ============================================================================

// saveScoresToStorage saves scorer results to storage for observability.
// Corresponds to TS: async function saveScoresToStorage({...})
func saveScoresToStorage(
	ctx context.Context,
	storage MastraStorage,
	scorerResults map[string]any,
	target any,
	item RunEvalsDataItem,
	mastra any,
) error {
	entityID := getTargetID(target)
	entityType := "AGENT"
	if _, isWorkflow := target.(Workflow); isWorkflow {
		entityType = "WORKFLOW"
	}

	// Check if results are flat or nested.
	_, hasWorkflow := scorerResults["workflow"]
	_, hasSteps := scorerResults["steps"]
	isNested := hasWorkflow || hasSteps

	if !isNested {
		// Flat scorer results.
		for scorerID, scoreResult := range scorerResults {
			if sr, ok := scoreResult.(map[string]any); ok {
				if _, hasScore := sr["score"]; hasScore {
					_ = saveSingleScore(ctx, storage, sr, scorerID, entityID, entityType, mastra, target, item)
				}
			}
		}
	} else {
		// Nested workflow/step results.
		if workflowResults, ok := scorerResults["workflow"].(map[string]any); ok {
			for scorerID, scoreResult := range workflowResults {
				if sr, ok := scoreResult.(map[string]any); ok {
					if _, hasScore := sr["score"]; hasScore {
						_ = saveSingleScore(ctx, storage, sr, scorerID, entityID, "WORKFLOW", mastra, target, item)
					}
				}
			}
		}
		if stepsResults, ok := scorerResults["steps"].(map[string]any); ok {
			for stepID, stepScorers := range stepsResults {
				if ssMap, ok := stepScorers.(map[string]any); ok {
					for scorerID, scoreResult := range ssMap {
						if sr, ok := scoreResult.(map[string]any); ok {
							if _, hasScore := sr["score"]; hasScore {
								_ = saveSingleScore(ctx, storage, sr, scorerID, stepID, "STEP", mastra, target, item)
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// saveSingleScore saves a single scorer result to storage.
// Corresponds to TS: async function saveSingleScore({...})
func saveSingleScore(
	ctx context.Context,
	storage MastraStorage,
	scoreResult map[string]any,
	scorerID string,
	entityID string,
	entityType string,
	mastra any,
	target any,
	item RunEvalsDataItem,
) error {
	// Try to find the scorer via mastra.getScorerById(scorerId).
	type hasScorerByID interface {
		GetScorerByID(id string) any
	}
	type hasListScorers interface {
		ListScorers() map[string]any
	}
	type hasLogger interface {
		GetLogger() any
	}

	var scorerInfo map[string]any

	// 1. Try mastra-registered scorer
	if m, ok := mastra.(hasScorerByID); ok {
		scorer := m.GetScorerByID(scorerID)
		if scorer != nil {
			scorerInfo = buildScorerInfo(scorer, scorerID)
		}
	}

	// 2. Fall back to target's scorers
	if scorerInfo == nil {
		if t, ok := target.(hasListScorers); ok {
			targetScorers := t.ListScorers()
			for _, scorerEntry := range targetScorers {
				if se, ok := scorerEntry.(map[string]any); ok {
					if innerScorer, ok := se["scorer"].(map[string]any); ok {
						if innerScorer["id"] == scorerID {
							scorerInfo = buildScorerInfo(innerScorer, scorerID)
							break
						}
					}
				}
			}
		}
	}

	// Default scorer info if not found
	if scorerInfo == nil {
		scorerInfo = map[string]any{
			"id":   scorerID,
			"name": scorerID,
		}
	}

	// Extract tracing context if available.
	// In TS: item.tracingContext?.currentSpan?.isValid checks span validity.
	// The Go Span interface doesn't have IsValid() — a non-nil span is valid.
	var traceID, spanID string
	if item.ObservabilityContext != nil {
		cs := item.ObservabilityContext.TracingCtx.CurrentSpan
		if cs != nil {
			spanID = cs.ID()
			traceID = cs.TraceID()
		}
	}

	// Build additional context with groundTruth if available
	var additionalContext map[string]any
	if item.GroundTruth != nil {
		additionalContext = map[string]any{
			"groundTruth": item.GroundTruth,
		}
	}

	// Build the payload
	payload := make(map[string]any)
	for k, v := range scoreResult {
		payload[k] = v
	}
	payload["scorerId"] = scorerID
	payload["entityId"] = entityID
	payload["entityType"] = entityType
	payload["source"] = "TEST"
	payload["scorer"] = scorerInfo

	// Entity info
	targetName := ""
	if a, ok := target.(Agent); ok {
		targetName = a.GetName()
	} else if w, ok := target.(Workflow); ok {
		targetName = w.GetName()
	}
	payload["entity"] = map[string]any{
		"id":   getTargetID(target),
		"name": targetName,
	}

	// Include requestContext from item
	if item.RequestContext != nil {
		payload["requestContext"] = item.RequestContext
	}

	// Include additionalContext with groundTruth
	if additionalContext != nil {
		payload["additionalContext"] = additionalContext
	}

	// Include tracing information
	if traceID != "" {
		payload["traceId"] = traceID
	}
	if spanID != "" {
		payload["spanId"] = spanID
	}

	// Call validateAndSaveScore via storage
	if err := validateAndSaveScore(ctx, storage, payload); err != nil {
		// Log error but don't fail the evaluation
		if m, ok := mastra.(hasLogger); ok {
			if log := m.GetLogger(); log != nil {
				type hasWarn interface {
					Warn(msg string, args ...any)
				}
				if w, ok := log.(hasWarn); ok {
					w.Warn(fmt.Sprintf("Failed to save score for scorer %s: %v", scorerID, err))
				}
			}
		}
	}

	return nil
}

// buildScorerInfo builds scorer info map from a scorer object.
func buildScorerInfo(scorer any, fallbackID string) map[string]any {
	info := map[string]any{
		"id":   fallbackID,
		"name": fallbackID,
	}
	if m, ok := scorer.(map[string]any); ok {
		if id, ok := m["id"].(string); ok {
			info["id"] = id
		}
		if name, ok := m["name"].(string); ok {
			info["name"] = name
		}
		if desc, ok := m["description"].(string); ok {
			info["description"] = desc
		}
		if t, ok := m["type"].(string); ok {
			info["type"] = t
		}
		if m["judge"] != nil {
			info["hasJudge"] = true
		}
	}
	type hasID interface{ ID() string }
	type hasName interface{ Name() string }
	if s, ok := scorer.(hasID); ok {
		info["id"] = s.ID()
	}
	if s, ok := scorer.(hasName); ok {
		info["name"] = s.Name()
	}
	return info
}

// validateAndSaveScore validates and saves a score payload to storage.
// Corresponds to TS: export async function validateAndSaveScore(storage, payload)
func validateAndSaveScore(ctx context.Context, storage MastraStorage, payload map[string]any) error {
	scoresStore, err := storage.GetStore("scores")
	if err != nil {
		return fmt.Errorf("scores storage domain is not available: %w", err)
	}
	if scoresStore == nil {
		return fmt.Errorf("scores storage domain is not available")
	}

	// Call saveScore on the scores store.
	type hasSaveScore interface {
		SaveScore(ctx context.Context, payload map[string]any) error
	}
	if ss, ok := scoresStore.(hasSaveScore); ok {
		return ss.SaveScore(ctx, payload)
	}

	return fmt.Errorf("scores store does not implement SaveScore")
}

// ============================================================================
// Helpers
// ============================================================================

// extractScoringData extracts the scoringData map from a target result.
func extractScoringData(targetResult any) map[string]any {
	if m, ok := targetResult.(map[string]any); ok {
		if sd, ok := m["scoringData"].(map[string]any); ok {
			return sd
		}
	}
	return map[string]any{}
}

// getMastraFromTarget extracts the Mastra instance from a target.
func getMastraFromTarget(target any) any {
	if a, ok := target.(Agent); ok {
		return a.GetMastraInstance()
	}
	if w, ok := target.(Workflow); ok {
		return w.GetMastra()
	}
	return nil
}

// getStorageFromMastra extracts the storage from a Mastra instance.
// STUB REASON: Uses runtime type assertion instead of importing core.Mastra
// to avoid adding a dependency on the core package from evals.
func getStorageFromMastra(mastra any) MastraStorage {
	type hasGetStorage interface {
		GetStorage() MastraStorage
	}
	if m, ok := mastra.(hasGetStorage); ok {
		return m.GetStorage()
	}
	return nil
}

// getTargetID returns the ID of a target (Agent or Workflow).
func getTargetID(target any) string {
	if a, ok := target.(Agent); ok {
		return a.GetID()
	}
	if w, ok := target.(Workflow); ok {
		return w.GetID()
	}
	return ""
}
