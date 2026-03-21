package agentembed

import internalagent "github.com/brainlet/brainkit/internal/embed/agent"

type ProviderConfig = internalagent.ProviderConfig
type Usage = internalagent.Usage
type FinishReason = internalagent.FinishReason

const (
	FinishStop          = internalagent.FinishStop
	FinishLength        = internalagent.FinishLength
	FinishContentFilter = internalagent.FinishContentFilter
	FinishToolCalls     = internalagent.FinishToolCalls
	FinishError         = internalagent.FinishError
	FinishSuspended     = internalagent.FinishSuspended
	FinishOther         = internalagent.FinishOther
)

type ResponseMeta = internalagent.ResponseMeta
type Message = internalagent.Message
type Tool = internalagent.Tool
type ToolContext = internalagent.ToolContext
type ToolChoice = internalagent.ToolChoice
type ToolCall = internalagent.ToolCall
type ToolResult = internalagent.ToolResult
type StepResult = internalagent.StepResult
type GenerateResult = internalagent.GenerateResult
type StreamResult = internalagent.StreamResult

func SystemMessage(content string) Message {
	return internalagent.SystemMessage(content)
}

func UserMessage(content string) Message {
	return internalagent.UserMessage(content)
}

func AssistantMessage(content string) Message {
	return internalagent.AssistantMessage(content)
}

func Float64(v float64) *float64 {
	return internalagent.Float64(v)
}

func Int(v int) *int {
	return internalagent.Int(v)
}
