// Ported from: packages/provider/src/errors/load-setting-error.ts
package errors

// LoadSettingError indicates a failure to load a setting.
type LoadSettingError struct {
	AISDKError
}

// NewLoadSettingError creates a new LoadSettingError.
func NewLoadSettingError(message string) *LoadSettingError {
	return &LoadSettingError{
		AISDKError: AISDKError{
			Name:    "AI_LoadSettingError",
			Message: message,
		},
	}
}

// IsLoadSettingError checks if an error is a LoadSettingError.
func IsLoadSettingError(err error) bool {
	var target *LoadSettingError
	return As(err, &target)
}
