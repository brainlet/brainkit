// Ported from: packages/core/src/stream/MastraWorkflowStream.ts
package stream

import (
	"strconv"
	"sync"
)

// MastraWorkflowStreamParams are the constructor parameters.
type MastraWorkflowStreamParams struct {
	// CreateStream creates the underlying stream. It receives a writable channel
	// for enqueuing processed chunks and returns a readable channel of raw chunks.
	CreateStream func(writer chan<- ChunkType) (<-chan ChunkType, error)
	// Run is the workflow run reference.
	Run *Run
}

// MastraWorkflowStream manages a workflow's stream, wrapping it with
// workflow-start and workflow-finish events and tracking token usage.
type MastraWorkflowStream struct {
	mu sync.Mutex

	usageCount LanguageModelUsage
	run        *Run

	// chunks is the output channel for consumers.
	chunks chan ChunkType

	// streamDone signals when the stream finishes.
	streamDone chan struct{}
	streamErr  error
}

// NewMastraWorkflowStream creates a new MastraWorkflowStream.
func NewMastraWorkflowStream(params MastraWorkflowStreamParams) *MastraWorkflowStream {
	s := &MastraWorkflowStream{
		usageCount: LanguageModelUsage{},
		run:        params.Run,
		chunks:     make(chan ChunkType, 256),
		streamDone: make(chan struct{}),
	}

	go s.start(params)
	return s
}

// start runs the stream processing pipeline.
func (s *MastraWorkflowStream) start(params MastraWorkflowStreamParams) {
	defer close(s.chunks)
	defer close(s.streamDone)

	writer := make(chan ChunkType, 256)

	// Forward writer chunks to output, tracking usage for finish events
	var writerWg sync.WaitGroup
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		for chunk := range writer {
			s.processWriterChunk(chunk)
			s.chunks <- chunk
		}
	}()

	// Emit workflow-start
	s.chunks <- ChunkType{
		BaseChunkType: BaseChunkType{
			RunID: params.Run.RunID,
			From:  ChunkFromWorkflow,
		},
		Type: "workflow-start",
		Payload: &WorkflowStartPayload{
			WorkflowID: params.Run.WorkflowID,
		},
	}

	stream, err := params.CreateStream(writer)
	if err != nil {
		s.streamErr = err
		close(writer)
		writerWg.Wait()
		return
	}

	workflowStatus := WorkflowRunStatusSuccess

	for chunk := range stream {
		switch chunk.Type {
		case "step-finish":
			if payload, ok := chunk.Payload.(*StepFinishPayload); ok {
				s.mu.Lock()
				s.updateUsageFromStepFinish(payload)
				s.mu.Unlock()
			}
		case "workflow-canceled":
			workflowStatus = WorkflowRunStatusCanceled
		case "workflow-step-suspended":
			workflowStatus = WorkflowRunStatusSuspended
		case "workflow-step-result":
			if payload, ok := chunk.Payload.(*WorkflowStepResultPayload); ok {
				if payload.Status == WorkflowStepStatusFailed {
					workflowStatus = WorkflowRunStatusFailed
				}
			}
		}

		s.chunks <- chunk
	}

	close(writer)
	writerWg.Wait()

	// Emit workflow-finish
	s.mu.Lock()
	usage := s.usageCount
	s.mu.Unlock()

	s.chunks <- ChunkType{
		BaseChunkType: BaseChunkType{
			RunID: params.Run.RunID,
			From:  ChunkFromWorkflow,
		},
		Type: "workflow-finish",
		Payload: &WorkflowFinishPayload{
			WorkflowStatus: workflowStatus,
			Output:         WorkflowFinishUsage{Usage: usage},
			Metadata:       map[string]any{},
		},
	}
}

// processWriterChunk handles usage tracking for chunks flowing through the writer.
func (s *MastraWorkflowStream) processWriterChunk(chunk ChunkType) {
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
		s.extractUsageFromFinish(output.Payload)
	}
}

// extractUsageFromFinish extracts usage from a finish payload.
func (s *MastraWorkflowStream) extractUsageFromFinish(payload any) {
	if payload == nil {
		return
	}
	m, ok := payload.(map[string]any)
	if !ok {
		return
	}
	if usage, ok := m["usage"]; ok {
		s.mu.Lock()
		s.updateUsageFromAny(usage)
		s.mu.Unlock()
	}
}

// updateUsageFromStepFinish extracts usage from a StepFinishPayload.
func (s *MastraWorkflowStream) updateUsageFromStepFinish(payload *StepFinishPayload) {
	usage := payload.Output.Usage
	s.usageCount.InputTokens += usage.InputTokens
	s.usageCount.OutputTokens += usage.OutputTokens
	s.usageCount.TotalTokens += usage.TotalTokens
}

// updateUsageFromAny parses usage from a loosely-typed value.
func (s *MastraWorkflowStream) updateUsageFromAny(usage any) {
	m, ok := usage.(map[string]any)
	if !ok {
		return
	}

	// Handle V2 format (inputTokens/outputTokens)
	if _, ok := m["inputTokens"]; ok {
		s.usageCount.InputTokens += parseTokenValueWS(m, "inputTokens")
		s.usageCount.OutputTokens += parseTokenValueWS(m, "outputTokens")
	} else if _, ok := m["promptTokens"]; ok {
		// Handle V1 format (promptTokens/completionTokens)
		s.usageCount.InputTokens += parseTokenValueWS(m, "promptTokens")
		s.usageCount.OutputTokens += parseTokenValueWS(m, "completionTokens")
	}
	s.usageCount.TotalTokens += parseTokenValueWS(m, "totalTokens")
}

// parseTokenValueWS is a local helper to parse token values (WS = workflow stream).
func parseTokenValueWS(m map[string]any, key string) int {
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

// Chunks returns the read-only channel of output chunks.
func (s *MastraWorkflowStream) Chunks() <-chan ChunkType {
	return s.chunks
}

// AwaitStatus blocks until the stream finishes and returns the execution status.
func (s *MastraWorkflowStream) AwaitStatus() (string, error) {
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
func (s *MastraWorkflowStream) AwaitResult() (*RunExecutionResult, error) {
	<-s.streamDone
	return s.run.GetExecutionResults()
}

// AwaitUsage blocks until the stream finishes and returns accumulated usage.
func (s *MastraWorkflowStream) AwaitUsage() LanguageModelUsage {
	<-s.streamDone
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usageCount
}
