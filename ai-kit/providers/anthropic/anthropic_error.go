// Ported from: packages/anthropic/src/anthropic-error.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// AnthropicErrorData represents the error data returned by the Anthropic API.
type AnthropicErrorData struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// anthropicErrorDataSchema is the schema for parsing Anthropic error responses.
var anthropicErrorDataSchema = providerutils.LazySchema(func() *providerutils.Schema[AnthropicErrorData] {
	return &providerutils.Schema[AnthropicErrorData]{}
})

// anthropicFailedResponseHandler handles failed API responses from Anthropic.
var anthropicFailedResponseHandler = providerutils.CreateJsonErrorResponseHandler(
	anthropicErrorDataSchema(),
	func(data AnthropicErrorData) string {
		return data.Error.Message
	},
	nil,
)
