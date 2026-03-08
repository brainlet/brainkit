// Ported from: packages/core/src/llm/model/registry-generator.ts
package model

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// FetchProvidersResult
// ---------------------------------------------------------------------------

// FetchProvidersResult holds the result of fetching providers from gateways.
type FetchProvidersResult struct {
	Providers map[string]ProviderConfig `json:"providers"`
	Models    map[string][]string       `json:"models"`
}

// ---------------------------------------------------------------------------
// AtomicWriteFile
// ---------------------------------------------------------------------------

// AtomicWriteFile writes a file atomically using the write-to-temp-then-rename pattern.
// This prevents file corruption when multiple processes write to the same file concurrently.
//
// The rename operation is atomic on POSIX systems when source and destination
// are on the same filesystem.
func AtomicWriteFile(filePath string, content string) error {
	// Create a unique temp file name using PID, timestamp, and random suffix to avoid collisions
	randomSuffix := randomAlphanumeric(13)
	tempPath := fmt.Sprintf("%s.%d.%d.%s.tmp", filePath, os.Getpid(), time.Now().UnixMilli(), randomSuffix)

	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return err
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		// Clean up temp file if it exists
		_ = os.Remove(tempPath)
		return err
	}

	return nil
}

// randomAlphanumeric generates a random alphanumeric string of the given length.
func randomAlphanumeric(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// FetchProvidersFromGateways
// ---------------------------------------------------------------------------

// FetchProvidersFromGateways fetches providers from all gateways with retry logic.
func FetchProvidersFromGateways(gateways []MastraModelGateway) (*FetchProvidersResult, error) {
	allProviders := make(map[string]ProviderConfig)
	allModels := make(map[string][]string)

	const maxRetries = 3

	for _, gateway := range gateways {
		var lastError error

		for attempt := 1; attempt <= maxRetries; attempt++ {
			providers, err := gateway.FetchProviders()
			if err != nil {
				lastError = err
				if attempt < maxRetries {
					// Wait before retrying (exponential backoff)
					delayMs := math.Min(1000*math.Pow(2, float64(attempt-1)), 5000)
					time.Sleep(time.Duration(delayMs) * time.Millisecond)
				}
				continue
			}

			// models.dev is a provider registry, not a true gateway -- don't prefix its providers
			isProviderRegistry := gateway.ID() == "models.dev"

			for providerID, config := range providers {
				// For true gateways, use gateway.id as prefix (e.g., "netlify/anthropic")
				// Special case: if providerId matches gateway.id, it's a unified gateway
				// (e.g., azure-openai returning {azure-openai: {...}})
				// In that case, use just the gateway ID to avoid duplication
				var typeProviderID string
				if isProviderRegistry {
					typeProviderID = providerID
				} else if providerID == gateway.ID() {
					typeProviderID = gateway.ID()
				} else {
					typeProviderID = gateway.ID() + "/" + providerID
				}

				allProviders[typeProviderID] = config

				// Sort models alphabetically for consistent ordering
				models := make([]string, len(config.Models))
				copy(models, config.Models)
				sort.Strings(models)
				allModels[typeProviderID] = models
			}

			lastError = nil
			break // Success, exit retry loop
		}

		if lastError != nil {
			return nil, lastError
		}
	}

	return &FetchProvidersResult{
		Providers: allProviders,
		Models:    allModels,
	}, nil
}

// ---------------------------------------------------------------------------
// GenerateTypesContent
// ---------------------------------------------------------------------------

// GenerateTypesContent generates TypeScript type definitions content from a models map.
// This is primarily useful for code generation tooling; in the Go port it produces
// the same TS output as the original for compatibility.
func GenerateTypesContent(models map[string][]string) string {
	// Sort providers for deterministic output
	providers := make([]string, 0, len(models))
	for p := range models {
		providers = append(providers, p)
	}
	sort.Strings(providers)

	var entries []string
	for _, provider := range providers {
		modelList := models[provider]
		modelStrings := make([]string, len(modelList))
		for i, m := range modelList {
			modelStrings[i] = "'" + m + "'"
		}

		// Quote provider key if it's not a valid JavaScript identifier
		providerKey := provider
		if !isValidJSIdentifier(provider) {
			providerKey = "'" + provider + "'"
		}

		// Format array based on length (prettier printWidth: 120)
		singleLine := fmt.Sprintf("  readonly %s: readonly [%s];", providerKey, strings.Join(modelStrings, ", "))

		if len(singleLine) > 120 {
			formattedModels := make([]string, len(modelList))
			for i, m := range modelList {
				formattedModels[i] = "    '" + m + "',"
			}
			entries = append(entries, fmt.Sprintf("  readonly %s: readonly [\n%s\n  ];", providerKey, strings.Join(formattedModels, "\n")))
		} else {
			entries = append(entries, singleLine)
		}
	}

	providerModelsEntries := strings.Join(entries, "\n")

	return fmt.Sprintf(`/**
 * THIS FILE IS AUTO-GENERATED - DO NOT EDIT
 * Generated from model gateway providers
 */

/**
 * Provider models mapping type
 * This is derived from the JSON data and provides type-safe access
 */
export type ProviderModelsMap = {
%s
};

/**
 * Union type of all registered provider IDs
 */
export type Provider = keyof ProviderModelsMap;

/**
 * Provider models mapping interface
 */
export interface ProviderModels {
  [key: string]: string[];
}

/**
 * OpenAI-compatible model ID type
 * Dynamically derived from ProviderModelsMap
 * Full provider/model paths (e.g., "openai/gpt-4o", "anthropic/claude-3-5-sonnet-20241022")
 */
export type ModelRouterModelId =
  | {
      [P in Provider]: `+"`${P}/${ProviderModelsMap[P][number]}`"+`;
    }[Provider]
  | (string & {});

/**
 * Extract the model part from a ModelRouterModelId for a specific provider
 * Dynamically derived from ProviderModelsMap
 * Example: ModelForProvider<'openai'> = 'gpt-4o' | 'gpt-4-turbo' | ...
 */
export type ModelForProvider<P extends Provider> = ProviderModelsMap[P][number];
`, providerModelsEntries)
}

// isValidJSIdentifier checks if a string is a valid JavaScript identifier.
func isValidJSIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, c := range s {
		if i == 0 {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$') {
				return false
			}
		} else {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$') {
				return false
			}
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// WriteRegistryFiles
// ---------------------------------------------------------------------------

// RegistryFileData is the JSON structure written to the registry file.
type RegistryFileData struct {
	Providers map[string]ProviderConfig `json:"providers"`
	Models    map[string][]string       `json:"models"`
	Version   string                    `json:"version"`
}

// WriteRegistryFiles writes registry files to disk (JSON and .d.ts).
func WriteRegistryFiles(
	jsonPath string,
	typesPath string,
	providers map[string]ProviderConfig,
	models map[string][]string,
) error {
	// 0. Ensure directories exist
	jsonDir := filepath.Dir(jsonPath)
	typesDir := filepath.Dir(typesPath)
	if err := os.MkdirAll(jsonDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", jsonDir, err)
	}
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", typesDir, err)
	}

	// 1. Write JSON file atomically to prevent corruption from concurrent writes
	registryData := RegistryFileData{
		Providers: providers,
		Models:    models,
		Version:   "1.0.0",
	}

	jsonBytes, err := json.MarshalIndent(registryData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry data: %w", err)
	}

	if err := AtomicWriteFile(jsonPath, string(jsonBytes)); err != nil {
		return fmt.Errorf("failed to write JSON registry file: %w", err)
	}

	// 2. Generate .d.ts file with type-only declarations (also atomic)
	typeContent := GenerateTypesContent(models)
	if err := AtomicWriteFile(typesPath, typeContent); err != nil {
		return fmt.Errorf("failed to write types file: %w", err)
	}

	return nil
}
