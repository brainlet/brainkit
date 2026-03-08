// Ported from: packages/core/src/vector/validation.ts
package vector

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
)

// createVectorErrorID generates a standardized error ID for vector operations.
// Ported from: packages/core/src/storage/utils.ts createVectorErrorId().
// Format: MASTRA_VECTOR_{STORE}_{OPERATION}_{STATUS} (all upper-snake-case).
func createVectorErrorID(store, operation, status string) string {
	return fmt.Sprintf("MASTRA_VECTOR_%s_%s_%s",
		toUpperSnakeCase(store),
		toUpperSnakeCase(operation),
		toUpperSnakeCase(status),
	)
}

// camelBoundary matches transitions from lowercase to uppercase (camelCase → camel_Case).
var camelBoundary = regexp.MustCompile(`([a-z])([A-Z])`)

// acronymBoundary matches transitions between consecutive uppercase and uppercase-then-lowercase
// (XMLParser → XML_Parser).
var acronymBoundary = regexp.MustCompile(`([A-Z])([A-Z][a-z])`)

// nonAlphanumeric matches any character that is not A-Z or 0-9.
var nonAlphanumeric = regexp.MustCompile(`[^A-Z0-9]+`)

// leadingTrailingUnderscores matches underscores at the start or end of a string.
var leadingTrailingUnderscores = regexp.MustCompile(`^_+|_+$`)

// toUpperSnakeCase converts a string to UPPER_SNAKE_CASE.
// Ported from: packages/core/src/storage/utils.ts toUpperSnakeCase().
func toUpperSnakeCase(s string) string {
	// Insert underscore before uppercase letters that follow lowercase letters.
	result := camelBoundary.ReplaceAllString(s, "${1}_${2}")
	// Insert underscore before uppercase letters followed by lowercase letters.
	result = acronymBoundary.ReplaceAllString(result, "${1}_${2}")
	// Convert to uppercase.
	result = strings.ToUpper(result)
	// Replace any non-alphanumeric characters with underscore.
	result = nonAlphanumeric.ReplaceAllString(result, "_")
	// Remove leading/trailing underscores.
	result = leadingTrailingUnderscores.ReplaceAllString(result, "")
	return result
}

// ValidateUpsertInput validates upsert input parameters.
//
// It checks that:
//   - vectors is non-nil and non-empty
//   - metadata length matches vectors length (if metadata is provided and non-empty)
//   - ids length matches vectors length (if ids is provided)
//
// Returns a *mastraerror.MastraError if validation fails, nil otherwise.
func ValidateUpsertInput(storeName string, vectors [][]float64, metadata []map[string]any, ids []string) error {
	// Validate vectors array is not empty.
	if len(vectors) == 0 {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       createVectorErrorID(storeName, "UPSERT", "EMPTY_VECTORS"),
			Domain:   mastraerror.ErrorDomainMastraVector,
			Category: mastraerror.ErrorCategoryUser,
			Details: map[string]any{
				"message": "Vectors array cannot be empty",
			},
		})
	}

	// Validate metadata length matches vectors length (skip if metadata is empty/not provided).
	if len(metadata) > 0 && len(metadata) != len(vectors) {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       createVectorErrorID(storeName, "UPSERT", "METADATA_LENGTH_MISMATCH"),
			Domain:   mastraerror.ErrorDomainMastraVector,
			Category: mastraerror.ErrorCategoryUser,
			Details: map[string]any{
				"message":        "Metadata array length must match vectors array length",
				"vectorsLength":  len(vectors),
				"metadataLength": len(metadata),
			},
		})
	}

	// Validate ids length matches vectors length.
	if len(ids) > 0 && len(ids) != len(vectors) {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       createVectorErrorID(storeName, "UPSERT", "IDS_LENGTH_MISMATCH"),
			Domain:   mastraerror.ErrorDomainMastraVector,
			Category: mastraerror.ErrorCategoryUser,
			Details: map[string]any{
				"message":       "IDs array length must match vectors array length",
				"vectorsLength": len(vectors),
				"idsLength":     len(ids),
			},
		})
	}

	return nil
}

// ValidateTopK validates the topK parameter for queries.
// topK must be a positive integer.
func ValidateTopK(storeName string, topK int) error {
	if topK <= 0 {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       createVectorErrorID(storeName, "QUERY", "INVALID_TOP_K"),
			Domain:   mastraerror.ErrorDomainMastraVector,
			Category: mastraerror.ErrorCategoryUser,
			Details: map[string]any{
				"message": "topK must be a positive integer",
				"topK":    topK,
			},
		})
	}
	return nil
}

// ValidateVectorValues validates vector components for NaN and Infinity values.
func ValidateVectorValues(storeName string, vectors [][]float64) error {
	for i, vec := range vectors {
		if vec == nil {
			return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       createVectorErrorID(storeName, "UPSERT", "INVALID_VECTOR"),
				Domain:   mastraerror.ErrorDomainMastraVector,
				Category: mastraerror.ErrorCategoryUser,
				Details: map[string]any{
					"message":     fmt.Sprintf("Vector at index %d is null or undefined", i),
					"vectorIndex": i,
				},
			})
		}

		for j, value := range vec {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       createVectorErrorID(storeName, "UPSERT", "INVALID_VECTOR_VALUE"),
					Domain:   mastraerror.ErrorDomainMastraVector,
					Category: mastraerror.ErrorCategoryUser,
					Details: map[string]any{
						"message":        fmt.Sprintf("Vector contains invalid value (NaN or Infinity) at position [%d][%d]", i, j),
						"vectorIndex":    i,
						"componentIndex": j,
						"value":          fmt.Sprintf("%v", value),
					},
				})
			}
		}
	}
	return nil
}

// ValidateUpsert validates all upsert inputs including optional vector value validation.
// It combines ValidateUpsertInput and ValidateVectorValues.
func ValidateUpsert(storeName string, vectors [][]float64, metadata []map[string]any, ids []string, validateValues bool) error {
	if err := ValidateUpsertInput(storeName, vectors, metadata, ids); err != nil {
		return err
	}

	if validateValues && len(vectors) > 0 {
		if err := ValidateVectorValues(storeName, vectors); err != nil {
			return err
		}
	}

	return nil
}
