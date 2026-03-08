// Ported from: packages/core/src/stream/MastraAgentNetworkStream.ts
package stream

import (
	"strconv"
	"sync"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported (network stream-specific)
// ---------------------------------------------------------------------------

// Run is a stub for ../workflows Run.
// Stub: workflows imports stream (circular dep); must remain local definition.
type Run struct {
	RunID      string
	WorkflowID string
	// GetExecutionResults returns the execution results of the workflow run.
	// This is an internal method.
	GetExecutionResults func() (*RunExecutionResult, error)
}

// RunExecutionResult is a stub for the result of Run._getExecutionResults().
// Stub: workflows imports stream (circular dep); must remain local definition.
type RunExecutionResult struct {
	Status string         `json:"status"`
	Result map[string]any `json:"result,omitempty"`
}

// ---------------------------------------------------------------------------
// MastraAgentNetworkStream
// ---------------------------------------------------------------------------

// MastraAgentNetworkStreamParams are the constructor parameters.
type MastraAgentNetworkStreamParams struct {
	// CreateStream creates the underlying stream. It receives a writable channel
	// for enqueuing processed chunks and returns a readable channel of raw chunks.
	CreateStream func(writer chan<- ChunkType) (<-chan ChunkType, error)
	// Run is the workflow run reference.
	Run *Run
}

// MastraAgentNetworkStream manages a network agent stream, tracking usage
// and structured output across nested workflow/agent executions.
type MastraAgentNetworkStream struct {
	mu sync.Mutex

	// RunID is the unique identifier for this run.
	RunID string

	usageCount LanguageModelUsage
	run        *Run

	// Output channels
	chunks chan ChunkType

	// streamPromise signals when the stream finishes processing.
	streamDone chan struct{}
	streamErr  error

	// objectPromise resolves to the structured output object (if any).
	objectDone    chan struct{}
	objectValue   any
	objectErr     error
	objectOnce    sync.Once

	// objectStream delivers partial structured output objects.
	objectStream chan any
}

// NewMastraAgentNetworkStream creates a new MastraAgentNetworkStream.
func NewMastraAgentNetworkStream(params MastraAgentNetworkStreamParams) *MastraAgentNetworkStream {
	s := &MastraAgentNetworkStream{
		RunID: params.Run.RunID,
		usageCount: LanguageModelUsage{},
		run:        params.Run,
		chunks:     make(chan ChunkType, 256),
		streamDone: make(chan struct{}),
		objectDone: make(chan struct{}),
		objectStream: make(chan any, 64),
	}

	go s.start(params.CreateStream)
	return s
}

// start runs the stream processing pipeline.
func (s *MastraAgentNetworkStream) start(createStream func(writer chan<- ChunkType) (<-chan ChunkType, error)) {
	defer close(s.chunks)
	defer close(s.streamDone)

	writer := make(chan ChunkType, 256)
	defer close(writer)

	// Start a goroutine to forward writer chunks to the output, tracking usage.
	var writerWg sync.WaitGroup
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		for chunk := range writer {
			s.processWriterChunk(chunk)
			s.chunks <- chunk
		}
	}()

	stream, err := createStream(writer)
	if err != nil {
		s.streamErr = err
		s.resolveObject(nil, err)
		return
	}

	objectResolved := false

	for chunk := range stream {
		if chunk.Type == "workflow-step-output" {
			inner := s.getInnerChunk(chunk)

			switch inner.Type {
			case "routing-agent-end", "agent-execution-end", "workflow-execution-end":
				s.extractUsageFromPayload(inner.Payload)

			case "network-object":
				if payload, ok := inner.Payload.(map[string]any); ok {
					if obj, ok := payload["object"]; ok {
						select {
						case s.objectStream <- obj:
						default:
						}
					}
				}
				s.chunks <- inner

			case "network-object-result":
				if !objectResolved {
					objectResolved = true
					var obj any
					if payload, ok := inner.Payload.(map[string]any); ok {
						obj = payload["object"]
					}
					s.resolveObject(obj, nil)
				}
				s.chunks <- inner

			case "network-execution-event-finish":
				s.mu.Lock()
				finishPayload := map[string]any{}
				if p, ok := inner.Payload.(map[string]any); ok {
					for k, v := range p {
						finishPayload[k] = v
					}
				}
				finishPayload["usage"] = s.usageCount
				s.mu.Unlock()

				modified := inner
				modified.Payload = finishPayload
				s.chunks <- modified

			default:
				s.chunks <- inner
			}
		}
	}

	// If no object was resolved, resolve with nil
	if !objectResolved {
		s.resolveObject(nil, nil)
	}

	writerWg.Wait()
}

// processWriterChunk handles usage tracking for chunks flowing through the writer.
func (s *MastraAgentNetworkStream) processWriterChunk(chunk ChunkType) {
	if chunk.Type != "step-output" {
		return
	}
	payload, ok := chunk.Payload.(*StepOutputPayload)
	if !ok {
		return
	}
	output, ok := payload.Output.(*ChunkType)
	if !ok {
		return
	}
	if (output.From == ChunkFromAgent || output.From == ChunkFromWorkflow) && output.Type == "finish" {
		s.extractUsageFromPayload(output.Payload)
	}
}

// getInnerChunk recursively unwraps workflow-step-output chunks.
func (s *MastraAgentNetworkStream) getInnerChunk(chunk ChunkType) ChunkType {
	if chunk.Type == "workflow-step-output" {
		if payload, ok := chunk.Payload.(*StepOutputPayload); ok {
			if inner, ok := payload.Output.(*ChunkType); ok {
				return s.getInnerChunk(*inner)
			}
		}
	}
	return chunk
}

// extractUsageFromPayload extracts and accumulates usage from a payload.
func (s *MastraAgentNetworkStream) extractUsageFromPayload(payload any) {
	if payload == nil {
		return
	}
	m, ok := payload.(map[string]any)
	if !ok {
		return
	}
	if usage, ok := m["usage"]; ok {
		s.updateUsageFromAny(usage)
	}
}

// updateUsageFromAny parses usage from a loosely-typed value.
func (s *MastraAgentNetworkStream) updateUsageFromAny(usage any) {
	m, ok := usage.(map[string]any)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usageCount.InputTokens += parseTokenValueNS(m, "inputTokens")
	s.usageCount.OutputTokens += parseTokenValueNS(m, "outputTokens")
	s.usageCount.TotalTokens += parseTokenValueNS(m, "totalTokens")
	s.usageCount.ReasoningTokens += parseTokenValueNS(m, "reasoningTokens")
	s.usageCount.CachedInputTokens += parseTokenValueNS(m, "cachedInputTokens")
}

// parseTokenValueNS is a local helper to parse token values (NS = network stream).
func parseTokenValueNS(m map[string]any, key string) int {
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

// resolveObject resolves the object promise.
func (s *MastraAgentNetworkStream) resolveObject(value any, err error) {
	s.objectOnce.Do(func() {
		s.objectValue = value
		s.objectErr = err
		close(s.objectStream)
		close(s.objectDone)
	})
}

// Chunks returns the read-only channel of output chunks.
func (s *MastraAgentNetworkStream) Chunks() <-chan ChunkType {
	return s.chunks
}

// AwaitStatus blocks until the stream finishes and returns the execution status.
func (s *MastraAgentNetworkStream) AwaitStatus() (string, error) {
	<-s.streamDone
	result, err := s.run.GetExecutionResults()
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	return result.Status, nil
}

// AwaitResult blocks until the stream finishes and returns the execution result.
func (s *MastraAgentNetworkStream) AwaitResult() (*RunExecutionResult, error) {
	<-s.streamDone
	return s.run.GetExecutionResults()
}

// AwaitUsage blocks until the stream finishes and returns accumulated usage.
func (s *MastraAgentNetworkStream) AwaitUsage() LanguageModelUsage {
	<-s.streamDone
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usageCount
}

// AwaitObject blocks until the structured output object is available.
// Returns (object, error). Object may be nil if no structured output was requested.
func (s *MastraAgentNetworkStream) AwaitObject() (any, error) {
	<-s.objectDone
	return s.objectValue, s.objectErr
}

// ObjectStream returns a read-only channel of partial structured output objects.
func (s *MastraAgentNetworkStream) ObjectStream() <-chan any {
	return s.objectStream
}
