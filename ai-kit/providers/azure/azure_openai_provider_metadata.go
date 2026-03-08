// Ported from: packages/azure/src/azure-openai-provider-metadata.ts
package azure

import "github.com/brainlet/brainkit/ai-kit/providers/openai"

// AzureResponsesProviderMetadata wraps the OpenAI ResponsesProviderMetadata
// under the "azure" key.
type AzureResponsesProviderMetadata struct {
	Azure openai.ResponsesProviderMetadata `json:"azure"`
}

// AzureResponsesReasoningProviderMetadata wraps the OpenAI
// ResponsesReasoningProviderMetadata under the "azure" key.
type AzureResponsesReasoningProviderMetadata struct {
	Azure openai.ResponsesReasoningProviderMetadata `json:"azure"`
}

// AzureResponsesTextProviderMetadata wraps the OpenAI
// ResponsesTextProviderMetadata under the "azure" key.
type AzureResponsesTextProviderMetadata struct {
	Azure openai.ResponsesTextProviderMetadata `json:"azure"`
}

// AzureResponsesSourceDocumentProviderMetadata wraps the OpenAI
// ResponsesSourceDocumentProviderMetadata under the "azure" key.
type AzureResponsesSourceDocumentProviderMetadata struct {
	Azure openai.ResponsesSourceDocumentProviderMetadata `json:"azure"`
}
