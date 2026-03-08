// Ported from: packages/anthropic/src/get-cache-control.ts
package anthropic

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// MaxCacheBreakpoints is the maximum number of cache breakpoints per request.
const MaxCacheBreakpoints = 4

// getCacheControl extracts cache_control from provider metadata.
// Allows both cacheControl and cache_control for flexibility.
func getCacheControl(providerMetadata shared.ProviderMetadata) *AnthropicCacheControl {
	if providerMetadata == nil {
		return nil
	}
	anthropicMeta, ok := providerMetadata["anthropic"]
	if !ok || anthropicMeta == nil {
		return nil
	}

	// Allow both cacheControl and cache_control
	var raw any
	if v, ok := anthropicMeta["cacheControl"]; ok {
		raw = v
	} else if v, ok := anthropicMeta["cache_control"]; ok {
		raw = v
	}

	if raw == nil {
		return nil
	}

	// Pass through value assuming it is of the correct type.
	// The Anthropic API will validate the value.
	if m, ok := raw.(map[string]any); ok {
		cc := &AnthropicCacheControl{}
		if t, ok := m["type"].(string); ok {
			cc.Type = t
		}
		if ttl, ok := m["ttl"].(string); ok {
			cc.TTL = &ttl
		}
		return cc
	}

	return nil
}

// CacheControlValidator tracks cache breakpoint count and warnings.
type CacheControlValidator struct {
	breakpointCount int
	warnings        []shared.Warning
}

// NewCacheControlValidator creates a new CacheControlValidator.
func NewCacheControlValidator() *CacheControlValidator {
	return &CacheControlValidator{}
}

// CacheControlContext describes the context for a cache control check.
type CacheControlContext struct {
	Type     string
	CanCache bool
}

// GetCacheControl validates and returns cache control for the given context.
func (v *CacheControlValidator) GetCacheControl(
	providerMetadata shared.ProviderMetadata,
	ctx CacheControlContext,
) *AnthropicCacheControl {
	cc := getCacheControl(providerMetadata)
	if cc == nil {
		return nil
	}

	// Validate that cache_control is allowed in this context
	if !ctx.CanCache {
		details := fmt.Sprintf("cache_control cannot be set on %s. It will be ignored.", ctx.Type)
		v.warnings = append(v.warnings, shared.UnsupportedWarning{
			Feature: "cache_control on non-cacheable context",
			Details: &details,
		})
		return nil
	}

	// Validate cache breakpoint limit
	v.breakpointCount++
	if v.breakpointCount > MaxCacheBreakpoints {
		details := fmt.Sprintf(
			"Maximum %d cache breakpoints exceeded (found %d). This breakpoint will be ignored.",
			MaxCacheBreakpoints, v.breakpointCount,
		)
		v.warnings = append(v.warnings, shared.UnsupportedWarning{
			Feature: "cacheControl breakpoint limit",
			Details: &details,
		})
		return nil
	}

	return cc
}

// GetCacheControlFromOptions extracts cache control from provider options (ProviderOptions).
func getCacheControlFromOptions(providerOptions shared.ProviderOptions) *AnthropicCacheControl {
	if providerOptions == nil {
		return nil
	}
	// ProviderOptions has the same structure as ProviderMetadata
	return getCacheControl(providerOptions)
}

// GetWarnings returns accumulated warnings.
func (v *CacheControlValidator) GetWarnings() []shared.Warning {
	return v.warnings
}
