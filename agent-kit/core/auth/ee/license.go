// Ported from: packages/core/src/auth/ee/license.ts
package ee

import (
	"log"
	"os"
	"sync"
	"time"
)

// LicenseInfo contains license information.
type LicenseInfo struct {
	// Valid indicates whether the license is valid.
	Valid bool `json:"valid"`
	// ExpiresAt is the license expiration date.
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	// Features are the features enabled by this license.
	Features []string `json:"features,omitempty"`
	// Organization is the organization name.
	Organization string `json:"organization,omitempty"`
	// Tier is the license tier: "standard" or "enterprise".
	Tier string `json:"tier,omitempty"`
}

var (
	cachedLicense  *LicenseInfo
	cacheTimestamp time.Time
	cacheTTL       = 60 * time.Second // 1 minute
	cacheMu        sync.Mutex
)

// ValidateLicense validates a license key and returns license information.
//
// Currently implements a simple check for the presence of the license key.
// In production, this would validate against a license server.
func ValidateLicense(licenseKey string) *LicenseInfo {
	key := licenseKey
	if key == "" {
		key = os.Getenv("MASTRA_EE_LICENSE")
	}

	if key == "" {
		return &LicenseInfo{Valid: false}
	}

	// TODO: Implement actual license validation
	// For now, any non-empty key is considered valid
	// In production, this would:
	// 1. Verify signature of the license key
	// 2. Check expiration date embedded in key
	// 3. Optionally validate against license server

	// Simple validation: key should be at least 32 characters
	if len(key) < 32 {
		return &LicenseInfo{Valid: false}
	}

	return &LicenseInfo{
		Valid:    true,
		Features: []string{"user", "session", "sso", "rbac", "acl"},
		Tier:     "enterprise",
	}
}

// IsLicenseValid checks if EE features are enabled (valid license or cache).
func IsLicenseValid() bool {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	now := time.Now()

	// Return cached result if still valid
	if cachedLicense != nil && now.Sub(cacheTimestamp) < cacheTTL {
		return cachedLicense.Valid
	}

	// Validate and cache
	cachedLicense = ValidateLicense("")
	cacheTimestamp = now

	if !cachedLicense.Valid && os.Getenv("MASTRA_EE_LICENSE") != "" {
		log.Println("[mastra/auth-ee] Invalid or expired EE license. EE features are disabled.")
	}

	return cachedLicense.Valid
}

// IsEELicenseValid is a deprecated alias for IsLicenseValid.
// Provided for backward compatibility.
var IsEELicenseValid = IsLicenseValid

// IsFeatureEnabled checks if a specific EE feature is enabled.
func IsFeatureEnabled(feature string) bool {
	if !IsLicenseValid() {
		return false
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()

	// If license is valid but no features array, all features are enabled
	if cachedLicense == nil || len(cachedLicense.Features) == 0 {
		return true
	}

	for _, f := range cachedLicense.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// GetLicenseInfo returns the current license information.
// Returns nil if not validated yet.
func GetLicenseInfo() *LicenseInfo {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	return cachedLicense
}

// ClearLicenseCache clears the license cache (useful for testing).
func ClearLicenseCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cachedLicense = nil
	cacheTimestamp = time.Time{}
}

// IsDevEnvironment checks if running in a development/testing environment.
// In dev, EE features work without a license per the ee/LICENSE terms.
func IsDevEnvironment() bool {
	if os.Getenv("MASTRA_DEV") == "true" || os.Getenv("MASTRA_DEV") == "1" {
		return true
	}
	nodeEnv := os.Getenv("NODE_ENV")
	return nodeEnv != "production" && nodeEnv != "prod"
}

// IsEEEnabled checks if EE features should be active.
// Returns true if running in dev/test environment (always allowed) or if a valid license is present.
func IsEEEnabled() bool {
	if IsDevEnvironment() {
		return true
	}
	return IsLicenseValid()
}
