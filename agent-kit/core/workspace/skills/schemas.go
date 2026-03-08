// Ported from: packages/core/src/workspace/skills/schemas.ts
package skills

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// =============================================================================
// Constants
// =============================================================================

// SkillLimits holds recommended limits from the Agent Skills spec.
var SkillLimits = struct {
	// MaxInstructionTokens is the recommended max tokens for instructions.
	MaxInstructionTokens int
	// MaxInstructionLines is the recommended max lines for SKILL.md.
	MaxInstructionLines int
	// MaxNameLength is the max characters for name field.
	MaxNameLength int
	// MaxDescriptionLength is the max characters for description field.
	MaxDescriptionLength int
	// MaxCompatibilityLength is the max characters for compatibility field.
	MaxCompatibilityLength int
}{
	MaxInstructionTokens:   5000,
	MaxInstructionLines:    500,
	MaxNameLength:          64,
	MaxDescriptionLength:   1024,
	MaxCompatibilityLength: 500,
}

// =============================================================================
// Validation Result
// =============================================================================

// SkillValidationResult holds the result of skill metadata validation.
type SkillValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// =============================================================================
// Field Validators
// =============================================================================

// skillNamePattern matches valid skill names: lowercase letters, numbers, hyphens.
var skillNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// validateSkillName validates skill name according to spec.
func validateSkillName(name interface{}) []string {
	var errors []string
	fieldPath := "name"

	s, ok := name.(string)
	if !ok {
		errors = append(errors, fmt.Sprintf("%s: Expected string, received %T", fieldPath, name))
		return errors
	}

	if len(s) == 0 {
		errors = append(errors, fmt.Sprintf("%s: Skill name cannot be empty", fieldPath))
		return errors
	}

	if len(s) > SkillLimits.MaxNameLength {
		errors = append(errors, fmt.Sprintf("%s: Skill name must be %d characters or less", fieldPath, SkillLimits.MaxNameLength))
	}

	if !skillNamePattern.MatchString(s) {
		errors = append(errors, fmt.Sprintf("%s: Skill name must contain only lowercase letters, numbers, and hyphens", fieldPath))
	}

	if strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-") {
		errors = append(errors, fmt.Sprintf("%s: Skill name must not start or end with a hyphen", fieldPath))
	}

	if strings.Contains(s, "--") {
		errors = append(errors, fmt.Sprintf("%s: Skill name must not contain consecutive hyphens", fieldPath))
	}

	return errors
}

// validateSkillDescription validates skill description according to spec.
func validateSkillDescription(description interface{}) []string {
	var errors []string
	fieldPath := "description"

	s, ok := description.(string)
	if !ok {
		errors = append(errors, fmt.Sprintf("%s: Expected string, received %T", fieldPath, description))
		return errors
	}

	if len(s) == 0 {
		errors = append(errors, fmt.Sprintf("%s: Skill description cannot be empty", fieldPath))
		return errors
	}

	if len(s) > SkillLimits.MaxDescriptionLength {
		errors = append(errors, fmt.Sprintf("%s: Skill description must be %d characters or less", fieldPath, SkillLimits.MaxDescriptionLength))
	}

	if len(strings.TrimSpace(s)) == 0 {
		errors = append(errors, fmt.Sprintf("%s: Skill description cannot be only whitespace", fieldPath))
	}

	return errors
}

// validateSkillLicense validates skill license (optional string).
func validateSkillLicense(license interface{}) []string {
	var errors []string
	fieldPath := "license"

	if license == nil {
		return errors
	}

	if _, ok := license.(string); !ok {
		errors = append(errors, fmt.Sprintf("%s: Expected string, received %T", fieldPath, license))
	}

	return errors
}

// validateSkillMetadataField validates skill metadata field (optional map).
func validateSkillMetadataField(metadata interface{}) []string {
	var errors []string
	fieldPath := "metadata"

	if metadata == nil {
		return errors
	}

	if _, ok := metadata.(map[string]interface{}); !ok {
		errors = append(errors, fmt.Sprintf("%s: Expected object, received %T", fieldPath, metadata))
	}

	return errors
}

// =============================================================================
// Validation Helpers
// =============================================================================

// estimateTokens provides a rough token estimate (words * 1.3).
func estimateTokens(text string) int {
	words := strings.Fields(text)
	return int(math.Ceil(float64(len(words)) * 1.3))
}

// countLines counts lines in text.
func countLines(text string) int {
	return len(strings.Split(text, "\n"))
}

// =============================================================================
// Main Validation Function
// =============================================================================

// ValidateSkillMetadata validates skill metadata with optional content warnings.
func ValidateSkillMetadata(metadata interface{}, dirName string, instructions string) SkillValidationResult {
	var errors []string
	var warnings []string

	// Check that metadata is a struct or map
	data, ok := metadata.(SkillMetadata)
	if !ok {
		// Try as map
		dataMap, ok2 := metadata.(map[string]interface{})
		if !ok2 {
			errors = append(errors, fmt.Sprintf("Expected object, received %T", metadata))
			return SkillValidationResult{Valid: false, Errors: errors, Warnings: warnings}
		}

		// Validate map fields
		errors = append(errors, validateSkillName(dataMap["name"])...)
		errors = append(errors, validateSkillDescription(dataMap["description"])...)
		errors = append(errors, validateSkillLicense(dataMap["license"])...)
		errors = append(errors, validateSkillMetadataField(dataMap["metadata"])...)

		// Check directory name match
		if dirName != "" {
			if name, ok := dataMap["name"].(string); ok && name != dirName {
				errors = append(errors, fmt.Sprintf("Skill name %q must match directory name %q", name, dirName))
			}
		}
	} else {
		// Validate struct fields
		errors = append(errors, validateSkillName(data.Name)...)
		errors = append(errors, validateSkillDescription(data.Description)...)
		if data.License != "" {
			errors = append(errors, validateSkillLicense(data.License)...)
		}
		if data.Metadata != nil {
			errors = append(errors, validateSkillMetadataField(data.Metadata)...)
		}

		// Check directory name match
		if dirName != "" && data.Name != dirName {
			errors = append(errors, fmt.Sprintf("Skill name %q must match directory name %q", data.Name, dirName))
		}
	}

	// Check instruction limits (warnings only)
	if instructions != "" {
		lineCount := countLines(instructions)
		tokenEstimate := estimateTokens(instructions)

		if lineCount > SkillLimits.MaxInstructionLines {
			warnings = append(warnings, fmt.Sprintf(
				"Instructions have %d lines (recommended: <%d). Consider moving content to references/.",
				lineCount, SkillLimits.MaxInstructionLines,
			))
		}

		if tokenEstimate > SkillLimits.MaxInstructionTokens {
			warnings = append(warnings, fmt.Sprintf(
				"Instructions have ~%d estimated tokens (recommended: <%d). Consider moving content to references/.",
				tokenEstimate, SkillLimits.MaxInstructionTokens,
			))
		}
	}

	return SkillValidationResult{
		Valid:    len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}
}
