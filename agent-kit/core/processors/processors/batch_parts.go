// Ported from: packages/core/src/processors/processors/batch-parts.ts
package concreteprocessors

import (
	"sync"
	"time"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// BatchPartsState
// ---------------------------------------------------------------------------

// BatchPartsState holds the batching state across stream chunks.
type BatchPartsState struct {
	Batch            []processors.ChunkType
	TimeoutTriggered bool

	mu       sync.Mutex
	timer    *time.Timer
	timerSet bool
}

// ---------------------------------------------------------------------------
// BatchPartsOptions
// ---------------------------------------------------------------------------

// BatchPartsOptions holds configuration for BatchPartsProcessor.
type BatchPartsOptions struct {
	// BatchSize is the number of parts to batch together before emitting.
	// Default: 5.
	BatchSize int

	// MaxWaitTime is the maximum time to wait before emitting a batch (in milliseconds).
	// If set, will emit the current batch even if it hasn't reached BatchSize.
	// Default: 0 (no timeout).
	MaxWaitTime int

	// EmitOnNonText emits immediately when a non-text part is encountered.
	// Default: true.
	EmitOnNonText bool
}

// ---------------------------------------------------------------------------
// BatchPartsProcessor
// ---------------------------------------------------------------------------

// BatchPartsProcessor batches multiple stream parts together to reduce stream overhead.
// Only implements ProcessOutputStream -- does not process final results.
type BatchPartsProcessor struct {
	processors.BaseProcessor
	options BatchPartsOptions
}

// NewBatchPartsProcessor creates a new BatchPartsProcessor.
func NewBatchPartsProcessor(opts *BatchPartsOptions) *BatchPartsProcessor {
	o := BatchPartsOptions{
		BatchSize:     5,
		EmitOnNonText: true,
	}
	if opts != nil {
		if opts.BatchSize > 0 {
			o.BatchSize = opts.BatchSize
		}
		o.MaxWaitTime = opts.MaxWaitTime
		o.EmitOnNonText = opts.EmitOnNonText
	}
	return &BatchPartsProcessor{
		BaseProcessor: processors.NewBaseProcessor("batch-parts", "Batch Parts"),
		options:       o,
	}
}

// flushBatch flushes the current batch and returns a combined chunk (or nil).
func (b *BatchPartsProcessor) flushBatch(state *BatchPartsState) *processors.ChunkType {
	if len(state.Batch) == 0 {
		return nil
	}

	// Clear any existing timeout.
	if state.timerSet && state.timer != nil {
		state.timer.Stop()
		state.timer = nil
		state.timerSet = false
	}

	// If we only have one part, return it directly.
	if len(state.Batch) == 1 {
		part := state.Batch[0]
		state.Batch = state.Batch[:0]
		return &part
	}

	// Combine multiple text chunks into a single text part.
	var textChunks []processors.ChunkType
	for _, part := range state.Batch {
		if part.Type == "text-delta" {
			textChunks = append(textChunks, part)
		}
	}

	if len(textChunks) > 0 {
		// Combine all text deltas.
		combinedText := ""
		for _, part := range textChunks {
			if payload, ok := part.Payload.(map[string]any); ok {
				if text, ok := payload["text"].(string); ok {
					combinedText += text
				}
			}
		}

		combinedChunk := processors.ChunkType{
			Type: "text-delta",
			Payload: map[string]any{
				"text": combinedText,
				"id":   "text-1",
			},
		}

		state.Batch = state.Batch[:0]
		return &combinedChunk
	}

	// If no text chunks, return the first non-text part.
	part := state.Batch[0]
	state.Batch = state.Batch[1:]
	return &part
}

// getOrInitState retrieves or initializes the BatchPartsState from the processor state map.
func (b *BatchPartsProcessor) getOrInitState(state map[string]any) *BatchPartsState {
	if s, ok := state["_batchState"].(*BatchPartsState); ok {
		return s
	}
	s := &BatchPartsState{
		Batch: make([]processors.ChunkType, 0),
	}
	state["_batchState"] = s
	return s
}

// ProcessOutputStream batches stream parts together to reduce overhead.
func (b *BatchPartsProcessor) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	part := args.Part
	bState := b.getOrInitState(args.State)

	bState.mu.Lock()
	defer bState.mu.Unlock()

	// Check if a timeout has triggered a flush.
	if bState.TimeoutTriggered && len(bState.Batch) > 0 {
		bState.TimeoutTriggered = false
		bState.Batch = append(bState.Batch, part)
		return b.flushBatch(bState), nil
	}

	// If it's a non-text part and we should emit immediately, flush the batch first.
	if b.options.EmitOnNonText && part.Type != "text-delta" {
		batchedChunk := b.flushBatch(bState)
		if batchedChunk != nil {
			return batchedChunk, nil
		}
		return &part, nil
	}

	// Add the part to the current batch.
	bState.Batch = append(bState.Batch, part)

	// Check if we should emit based on batch size.
	if len(bState.Batch) >= b.options.BatchSize {
		return b.flushBatch(bState), nil
	}

	// Set up timeout for max wait time if specified.
	if b.options.MaxWaitTime > 0 && !bState.timerSet {
		bState.timerSet = true
		bState.timer = time.AfterFunc(time.Duration(b.options.MaxWaitTime)*time.Millisecond, func() {
			bState.mu.Lock()
			defer bState.mu.Unlock()
			bState.TimeoutTriggered = true
			bState.timerSet = false
			bState.timer = nil
		})
	}

	// Don't emit this part yet -- it's batched.
	return nil, nil
}

// Flush forces a flush of any remaining batched parts.
// This should be called when the stream ends to ensure no parts are lost.
func (b *BatchPartsProcessor) Flush(state map[string]any) *processors.ChunkType {
	bState := b.getOrInitState(state)
	bState.mu.Lock()
	defer bState.mu.Unlock()
	return b.flushBatch(bState)
}

// ProcessInput is not implemented for this processor.
func (b *BatchPartsProcessor) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	return nil, nil, nil, nil
}

// ProcessInputStep is not implemented for this processor.
func (b *BatchPartsProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ProcessOutputResult is not implemented for this processor.
func (b *BatchPartsProcessor) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (b *BatchPartsProcessor) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}
