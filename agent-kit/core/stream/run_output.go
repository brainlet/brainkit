// Ported from: packages/core/src/stream/RunOutput.ts
package stream

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
)

// ---------------------------------------------------------------------------
// DelayedPromise — Go equivalent of the TS DelayedPromise<T>
// ---------------------------------------------------------------------------

// DelayedPromiseStatus represents the state of a delayed promise.
type DelayedPromiseStatus string

const (
	DelayedPromiseStatusPending  DelayedPromiseStatus = "pending"
	DelayedPromiseStatusResolved DelayedPromiseStatus = "resolved"
	DelayedPromiseStatusRejected DelayedPromiseStatus = "rejected"
)

// DelayedPromise is the Go equivalent of the TS DelayedPromise<T>.
// It lazily creates a channel-based future that resolves or rejects.
type DelayedPromise[T any] struct {
	mu     sync.Mutex
	status DelayedPromiseStatus
	value  T
	err    error
	ch     chan struct{} // closed when resolved/rejected
}

// NewDelayedPromise creates a new pending DelayedPromise.
func NewDelayedPromise[T any]() *DelayedPromise[T] {
	return &DelayedPromise[T]{
		status: DelayedPromiseStatusPending,
		ch:     make(chan struct{}),
	}
}

// Resolve resolves the promise with a value.
func (dp *DelayedPromise[T]) Resolve(value T) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	if dp.status != DelayedPromiseStatusPending {
		return
	}
	dp.status = DelayedPromiseStatusResolved
	dp.value = value
	close(dp.ch)
}

// Reject rejects the promise with an error.
func (dp *DelayedPromise[T]) Reject(err error) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	if dp.status != DelayedPromiseStatusPending {
		return
	}
	dp.status = DelayedPromiseStatusRejected
	dp.err = err
	close(dp.ch)
}

// Await blocks until the promise is resolved or rejected. Returns (value, error).
func (dp *DelayedPromise[T]) Await() (T, error) {
	<-dp.ch
	dp.mu.Lock()
	defer dp.mu.Unlock()
	if dp.status == DelayedPromiseStatusRejected {
		var zero T
		return zero, dp.err
	}
	return dp.value, nil
}

// Status returns the current status of the promise.
func (dp *DelayedPromise[T]) Status() DelayedPromiseStatus {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	return dp.status
}

// ---------------------------------------------------------------------------
// WorkflowRunOutput
// ---------------------------------------------------------------------------

// WorkflowResult is a stub for ../workflows WorkflowResult.
// Stub: workflows imports stream (circular dep); must remain local definition.
type WorkflowResult = map[string]any

// WorkflowRunOutput manages a workflow run's stream, accumulating usage
// and tracking status. It implements the MastraBaseStream interface.
type WorkflowRunOutput struct {
	mu sync.Mutex

	// RunID is the unique identifier for this workflow run.
	RunID string
	// WorkflowID is the unique identifier for this workflow.
	WorkflowID string

	status         WorkflowRunStatus
	tripwireData   *StepTripwireData
	usageCount     LanguageModelUsage
	streamFinished bool
	streamError    error

	// bufferedChunks stores all emitted chunks for replay.
	bufferedChunks []WorkflowStreamEvent

	// subscribers receive new chunks in real time.
	subscribers []chan WorkflowStreamEvent
	// finishCh is closed when the stream finishes.
	finishCh chan struct{}

	consumeOnce sync.Once

	usagePromise  *DelayedPromise[LanguageModelUsage]
	resultPromise *DelayedPromise[WorkflowResult]
}

// WorkflowRunOutputParams are the constructor parameters.
type WorkflowRunOutputParams struct {
	RunID      string
	WorkflowID string
	Stream     <-chan WorkflowStreamEvent
}

// NewWorkflowRunOutput creates a new WorkflowRunOutput and starts consuming
// the input stream in a background goroutine.
func NewWorkflowRunOutput(params WorkflowRunOutputParams) *WorkflowRunOutput {
	w := &WorkflowRunOutput{
		RunID:         params.RunID,
		WorkflowID:    params.WorkflowID,
		status:        WorkflowRunStatusRunning,
		usageCount:    LanguageModelUsage{},
		finishCh:      make(chan struct{}),
		usagePromise:  NewDelayedPromise[LanguageModelUsage](),
		resultPromise: NewDelayedPromise[WorkflowResult](),
	}

	go w.consumeInput(params.Stream)
	return w
}

// consumeInput reads from the input stream, buffers chunks, and notifies subscribers.
func (w *WorkflowRunOutput) consumeInput(stream <-chan WorkflowStreamEvent) {
	// Emit workflow-start
	startChunk := ChunkType{
		BaseChunkType: BaseChunkType{
			RunID: w.RunID,
			From:  ChunkFromWorkflow,
		},
		Type: "workflow-start",
		Payload: &WorkflowStartPayload{
			WorkflowID: w.WorkflowID,
		},
	}
	w.emitChunk(startChunk)

	for chunk := range stream {
		if chunk.Type != "workflow-step-finish" {
			w.emitChunk(chunk)
		}
		w.processChunk(chunk)
	}

	// Stream closed — finalize
	w.mu.Lock()
	if w.status == WorkflowRunStatusRunning {
		w.status = WorkflowRunStatusSuccess
	}

	finishMetadata := map[string]any{}
	if w.streamError != nil {
		finishMetadata["error"] = w.streamError
		finishMetadata["errorMessage"] = w.streamError.Error()
	}

	finishPayload := &WorkflowFinishPayload{
		WorkflowStatus: w.status,
		Output:         WorkflowFinishUsage{Usage: w.usageCount},
		Metadata:       finishMetadata,
	}
	if w.status == WorkflowRunStatusTripwire && w.tripwireData != nil {
		finishPayload.Tripwire = w.tripwireData
	}

	finishChunk := ChunkType{
		BaseChunkType: BaseChunkType{
			RunID: w.RunID,
			From:  ChunkFromWorkflow,
		},
		Type:    "workflow-finish",
		Payload: finishPayload,
	}
	w.mu.Unlock()

	w.emitChunk(finishChunk)

	w.usagePromise.Resolve(w.usageCount)

	// Reject any unresolved promises
	if w.resultPromise.Status() == DelayedPromiseStatusPending {
		w.resultPromise.Reject(errors.New("promise 'result' was not resolved or rejected when stream finished"))
	}

	w.mu.Lock()
	w.streamFinished = true
	// Close all subscriber channels
	for _, sub := range w.subscribers {
		close(sub)
	}
	w.subscribers = nil
	close(w.finishCh)
	w.mu.Unlock()
}

// processChunk handles status changes and usage accumulation.
func (w *WorkflowRunOutput) processChunk(chunk WorkflowStreamEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	switch chunk.Type {
	case "workflow-step-output":
		w.processStepOutput(chunk)
	case "workflow-canceled":
		w.status = WorkflowRunStatusCanceled
	case "workflow-step-suspended":
		w.status = WorkflowRunStatusSuspended
	case "workflow-step-result":
		if payload, ok := chunk.Payload.(*WorkflowStepResultPayload); ok {
			if payload.Status == WorkflowStepStatusFailed {
				if payload.Tripwire != nil {
					w.status = WorkflowRunStatusTripwire
					w.tripwireData = payload.Tripwire
				} else {
					w.status = WorkflowRunStatusFailed
				}
			}
		}
	case "workflow-paused":
		w.status = WorkflowRunStatusPaused
	}
}

// processStepOutput extracts usage from step-output chunks.
func (w *WorkflowRunOutput) processStepOutput(chunk WorkflowStreamEvent) {
	if payload, ok := chunk.Payload.(*StepOutputPayload); ok {
		if output, ok := payload.Output.(*ChunkType); ok && output.Type == "finish" {
			w.extractUsageFromFinishPayload(output.Payload)
		}
	}
}

// extractUsageFromFinishPayload tries to find usage data in finish payloads.
func (w *WorkflowRunOutput) extractUsageFromFinishPayload(payload any) {
	if payload == nil {
		return
	}
	// Try map-based payload (common for loosely-typed payloads)
	if m, ok := payload.(map[string]any); ok {
		if usage, ok := m["usage"]; ok {
			w.updateUsageFromAny(usage)
			return
		}
		if output, ok := m["output"]; ok {
			if om, ok := output.(map[string]any); ok {
				if usage, ok := om["usage"]; ok {
					w.updateUsageFromAny(usage)
				}
			}
		}
	}
}

// emitChunk buffers a chunk and sends it to all current subscribers.
func (w *WorkflowRunOutput) emitChunk(chunk WorkflowStreamEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.bufferedChunks = append(w.bufferedChunks, chunk)
	for _, sub := range w.subscribers {
		// Non-blocking send; if subscriber can't keep up, chunk is dropped.
		select {
		case sub <- chunk:
		default:
		}
	}
}

// updateUsageFromAny parses usage from a loosely-typed value.
func (w *WorkflowRunOutput) updateUsageFromAny(usage any) {
	m, ok := usage.(map[string]any)
	if !ok {
		return
	}
	w.usageCount.InputTokens += parseTokenValue(m, "inputTokens")
	if w.usageCount.InputTokens == 0 {
		// V1 format
		w.usageCount.InputTokens += parseTokenValue(m, "promptTokens")
	}
	w.usageCount.OutputTokens += parseTokenValue(m, "outputTokens")
	if w.usageCount.OutputTokens == 0 {
		w.usageCount.OutputTokens += parseTokenValue(m, "completionTokens")
	}
	w.usageCount.TotalTokens += parseTokenValue(m, "totalTokens")
	w.usageCount.ReasoningTokens += parseTokenValue(m, "reasoningTokens")
	w.usageCount.CachedInputTokens += parseTokenValue(m, "cachedInputTokens")
}

// parseTokenValue extracts an integer token count from a map value (handles string or number).
func parseTokenValue(m map[string]any, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

// UpdateResults resolves the result promise with the given results.
// This is an internal method.
func (w *WorkflowRunOutput) UpdateResults(results WorkflowResult) {
	w.resultPromise.Resolve(results)
}

// RejectResults rejects the result promise and marks the workflow as failed.
// This is an internal method.
func (w *WorkflowRunOutput) RejectResults(err error) {
	w.resultPromise.Reject(err)
	w.mu.Lock()
	w.status = WorkflowRunStatusFailed
	w.streamError = err
	w.mu.Unlock()
}

// Resume replaces the underlying stream with a new one and resets state.
// This is an internal method.
func (w *WorkflowRunOutput) Resume(stream <-chan WorkflowStreamEvent) {
	w.mu.Lock()
	w.streamFinished = false
	w.status = WorkflowRunStatusRunning
	w.usagePromise = NewDelayedPromise[LanguageModelUsage]()
	w.resultPromise = NewDelayedPromise[WorkflowResult]()
	w.finishCh = make(chan struct{})
	w.consumeOnce = sync.Once{}
	w.mu.Unlock()

	go w.consumeInput(stream)
}

// ConsumeStream reads through the full stream to drive processing.
// Safe to call multiple times; only the first call actually consumes.
func (w *WorkflowRunOutput) ConsumeStream(onError func(error)) {
	w.consumeOnce.Do(func() {
		go func() {
			ch := w.FullStream()
			for range ch {
				// drain
			}
		}()
	})
}

// FullStream returns a channel that replays all buffered chunks and then
// delivers new chunks in real time until the stream finishes.
func (w *WorkflowRunOutput) FullStream() <-chan WorkflowStreamEvent {
	out := make(chan WorkflowStreamEvent, 256)
	go func() {
		defer close(out)

		w.mu.Lock()
		// Replay buffered chunks
		for _, chunk := range w.bufferedChunks {
			out <- chunk
		}

		if w.streamFinished {
			w.mu.Unlock()
			return
		}

		// Subscribe for new chunks
		sub := make(chan WorkflowStreamEvent, 256)
		w.subscribers = append(w.subscribers, sub)
		w.mu.Unlock()

		for chunk := range sub {
			out <- chunk
		}
	}()
	return out
}

// GetStatus returns the current workflow run status.
func (w *WorkflowRunOutput) GetStatus() WorkflowRunStatus {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

// AwaitResult blocks until the result promise is resolved. Returns (result, error).
func (w *WorkflowRunOutput) AwaitResult() (WorkflowResult, error) {
	return w.resultPromise.Await()
}

// AwaitUsage blocks until the usage promise is resolved. Returns (usage, error).
func (w *WorkflowRunOutput) AwaitUsage() (LanguageModelUsage, error) {
	return w.usagePromise.Await()
}

// WaitForFinish blocks until the stream has fully finished.
func (w *WorkflowRunOutput) WaitForFinish() {
	<-w.finishCh
}

// suppress unused import warning for log and fmt
var _ = log.Println
var _ = fmt.Sprintf
