// Ported from: packages/provider-utils/src/load-optional-setting.ts
package providerutils

import "os"

// LoadOptionalSettingOptions are the options for LoadOptionalSetting.
type LoadOptionalSettingOptions struct {
	// SettingValue is the direct setting value.
	SettingValue *string
	// EnvironmentVariableName is the name of the environment variable.
	EnvironmentVariableName string
}

// LoadOptionalSetting loads an optional string setting from a parameter or environment variable.
// Returns nil if the setting is not found.
func LoadOptionalSetting(opts LoadOptionalSettingOptions) *string {
	if opts.SettingValue != nil {
		return opts.SettingValue
	}

	val := os.Getenv(opts.EnvironmentVariableName)
	if val == "" {
		return nil
	}

	return &val
}
